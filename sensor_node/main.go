package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/sensor_node/proto"
)

const (
	serverName = "localhost"
	maxRetries = 5
	baseDelay  = 100 * time.Millisecond
	maxDelay   = 10 * time.Second
)

// Config holds the configuration for the sensor node
type Config struct {
	Rate       float64
	SensorName string
	SinkAddr   string

	UseTLS         bool
	CertFile       string
	ClientCertFile string
	ClientKeyFile  string
}

// SensorNode represents a sensor node that generates and sends data
type SensorNode struct {
	config Config
	client pb.TelemetryServiceClient
	conn   *grpc.ClientConn
	done   chan struct{}
}

func main() {
	config := parseFlags()

	node, err := NewSensorNode(config)
	if err != nil {
		log.Fatalf("Failed to create sensor node: %v", err)
	}
	defer node.Close()

	log.Printf(
		"Starting sensor node: %s, rate: %.2f msg/s, sink: %s, use TLS: %v, cert file: %s",
		config.SensorName,
		config.Rate,
		config.SinkAddr,
		config.UseTLS,
		config.CertFile,
	)
	log.Printf(
		"TLS: %v, Client cert: %s, Client key: %s",
		config.UseTLS,
		config.ClientCertFile,
		config.ClientKeyFile,
	)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping sensor node")
		node.Stop()
	}()

	node.Run()
}

func parseFlags() Config {
	var config Config

	flag.Float64Var(&config.Rate, "rate", 1.0, "Number of messages per second")
	flag.StringVar(&config.SensorName, "sensor-name", "default-sensor", "Name of the sensor")
	flag.StringVar(&config.SinkAddr, "sink-addr", "localhost:9090", "Address of the telemetry sink")

	flag.BoolVar(&config.UseTLS, "tls", false, "Use TLS for connection")
	flag.StringVar(&config.CertFile, "cert-file", "", "Path to TLS certificate file (optional)")
	flag.StringVar(&config.ClientCertFile, "client-cert", "", "Path to client certificate file (for mTLS)")
	flag.StringVar(&config.ClientKeyFile, "client-key", "", "Path to client private key file (for mTLS)")

	flag.Parse()

	return config
}

func NewSensorNode(config Config) (*SensorNode, error) {
	var opts []grpc.DialOption

	if config.UseTLS && config.CertFile != "" {
		creds, err := loadTLSCredentials(config)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))

		if config.ClientCertFile != "" && config.ClientKeyFile != "" {
			log.Println("mTLS enabled for client connection")
		} else {
			log.Println("TLS enabled for client connection")
		}
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(config.SinkAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sink: %w", err)
	}

	client := pb.NewTelemetryServiceClient(conn)

	return &SensorNode{
		config: config,
		client: client,
		conn:   conn,
		done:   make(chan struct{}),
	}, nil
}

func loadTLSCredentials(config Config) (credentials.TransportCredentials, error) {
	tlsConfig := &tls.Config{
		ServerName: serverName,
	}

	if config.CertFile != "" {
		caCert, err := os.ReadFile(config.CertFile)
		if err != nil {
			return nil, fmt.Errorf("read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("append CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// mTLS
	if config.ClientCertFile != "" && config.ClientKeyFile != "" {
		clientCert, err := tls.LoadX509KeyPair(config.ClientCertFile, config.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client certificates: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return credentials.NewTLS(tlsConfig), nil
}

func (s *SensorNode) Run() {
	interval := time.Duration(float64(time.Second) / s.config.Rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.generateAndSendData()
		case <-s.done:
			log.Println("Sensor node stopped")
			return
		}
	}
}

func (s *SensorNode) generateAndSendData() {
	sensorData := &pb.SensorData{
		SensorName:  s.config.SensorName,
		SensorValue: rand.Int31n(100),
		Timestamp:   timestamppb.Now(),
	}

	err := s.sendWithRetry(sensorData)
	if err != nil {
		log.Printf("Failed to send data after retries: %v", err)
	}
}

func (s *SensorNode) sendWithRetry(sensorData *pb.SensorData) error {
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		response, err := s.client.SendSensorData(ctx, sensorData)
		cancel()

		if err == nil {
			log.Printf("Sent: %s=%d at %s, Response: %s",
				sensorData.SensorName,
				sensorData.SensorValue,
				sensorData.Timestamp.AsTime().Format(time.RFC3339),
				response.Message)
			return nil
		}

		// Check if error is retryable
		if !s.isRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		if attempt < maxRetries-1 {
			delay := s.calculateDelay(attempt, baseDelay, maxDelay)
			log.Printf("Attempt %d failed: %v. Retrying in %v...", attempt+1, err, delay)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("max retries (%d) exceeded", maxRetries)
}

func (s *SensorNode) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return true
	}

	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted:
		return true
	case codes.InvalidArgument, codes.NotFound, codes.PermissionDenied, codes.Unauthenticated:
		return false
	default:
		return true
	}
}

func (s *SensorNode) calculateDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))

	jitter := time.Duration(rand.Float64()*0.5-0.25) * delay
	delay += jitter

	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

func (s *SensorNode) Stop() {
	close(s.done)
}

func (s *SensorNode) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}
