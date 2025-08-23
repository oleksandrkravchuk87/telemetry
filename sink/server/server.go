package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/sink/config"
	"github.com/sink/encryptor"
	pb "github.com/sink/proto"
	"github.com/sink/ratelimit"
)

const serverName = "localhost"

type SinkServer struct {
	pb.UnimplementedTelemetryServiceServer
	config      config.Config
	buffer      []byte
	bufferMutex sync.Mutex
	logFile     *os.File
	fileWriter  *bufio.Writer
	rateLimiter *ratelimit.RateLimiter
	encryptor   *encryption.AESGCMEncryptor
	done        chan struct{}
	wg          sync.WaitGroup
}

func NewSinkServer(config config.Config) (*SinkServer, error) {
	logFile, err := os.OpenFile(config.LogFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	fileWriter := bufio.NewWriter(logFile)

	var encryptor *encryption.AESGCMEncryptor
	if config.EnableEncryption {
		encryptor, err = encryption.NewAESGCMEncryptor(config.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create encryptor: %w", err)
		}
		log.Println("Log encryption enabled")
	}

	return &SinkServer{
		config:      config,
		buffer:      make([]byte, 0, config.BufferSize),
		logFile:     logFile,
		fileWriter:  fileWriter,
		rateLimiter: ratelimit.NewRateLimiter(config.RateLimit),
		encryptor:   encryptor,
		done:        make(chan struct{}),
	}, nil
}

func (s *SinkServer) Start() error {
	lis, err := net.Listen("tcp", s.config.BindAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	var opts []grpc.ServerOption

	if s.config.UseTLS {
		creds, err := s.loadTLSCredentials()
		if err != nil {
			return fmt.Errorf("load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
		if s.config.CAFile != "" {
			log.Println("mTLS enabled for gRPC server")
		} else {
			log.Println("TLS enabled for gRPC server")
		}
	}

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterTelemetryServiceServer(grpcServer, s)

	s.wg.Add(1)
	go s.flushTimer()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Server: %v", err)
		}
	}()

	<-s.done

	log.Println("Shutting down server...")
	grpcServer.GracefulStop()
	s.wg.Wait()

	return nil
}

func (s *SinkServer) loadTLSCredentials() (credentials.TransportCredentials, error) {
	serverCert, err := tls.LoadX509KeyPair(s.config.CertFile, s.config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ServerName:   serverName,
	}

	if s.config.CAFile != "" {
		caCert, err := os.ReadFile(s.config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("append CA certificate")
		}

		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.ClientCAs = caCertPool
	}

	return credentials.NewTLS(tlsConfig), nil
}

func (s *SinkServer) SendSensorData(ctx context.Context, req *pb.SensorData) (*pb.SensorDataResponse, error) {
	err := s.validateClientCertificateIfMTLS(ctx)
	if err != nil {
		log.Printf("Client certificate validation failed: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "invalid client certificate: %v", err)
	}

	data, err := proto.Marshal(req)
	if err != nil {
		log.Printf("marshal req: %v", err)
		return nil, status.Errorf(codes.Internal, "marshal data: %v", err)
	}

	if !s.rateLimiter.Allow(len(data)) {
		log.Printf("rate limit exceeded, dropping message from %s", req.SensorName)
		return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}

	logEntry := map[string]interface{}{
		"timestamp":    time.Now().UTC(),
		"sensor_name":  req.SensorName,
		"sensor_value": req.SensorValue,
		"data_time":    req.Timestamp.AsTime().UTC(),
	}

	logData, err := json.Marshal(logEntry)
	if err != nil {
		log.Printf("failed to marshal log entry: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to marshal log entry: %v", err)
	}

	if s.encryptor != nil {
		encryptedData, err := s.encryptor.Encrypt(logData)
		if err != nil {
			log.Printf("failed to encrypt log data: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to encrypt log data: %v", err)
		}

		logData = []byte(base64.StdEncoding.EncodeToString(encryptedData))
	}

	logData = append(logData, '\n')

	s.bufferMutex.Lock()

	if len(s.buffer)+len(logData) > s.config.BufferSize {
		log.Printf("flushing buffer due to size limit, max size: %d bytes", s.config.BufferSize)
		if err := s.flushBuffer(); err != nil {
			s.bufferMutex.Unlock()
			log.Printf("failed to flush buffer: %v", err)
			return nil, status.Errorf(codes.Internal, "flush buffer: %v", err)
		}
	}

	s.buffer = append(s.buffer, logData...)
	s.bufferMutex.Unlock()

	log.Printf("Received data from %s: value=%d", req.SensorName, req.SensorValue)

	return &pb.SensorDataResponse{
		Message: "Received successfully",
	}, nil
}

func (s *SinkServer) validateClientCertificateIfMTLS(ctx context.Context) error {
	if !s.config.UseTLS || s.config.CAFile == "" {
		return nil
	}

	peer, ok := peer.FromContext(ctx)
	if !ok {
		return fmt.Errorf("get peer from context")
	}

	tlsInfo, ok := peer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return fmt.Errorf("get TLS info")
	}

	if len(tlsInfo.State.PeerCertificates) == 0 {
		return fmt.Errorf("no client certificate provided")
	}

	clientCert := tlsInfo.State.PeerCertificates[0]
	log.Printf(
		"Client authenticated with certificate: Subject=%s, Issuer=%s",
		clientCert.Subject,
		clientCert.Issuer,
	)

	return nil
}

func (s *SinkServer) flushTimer() {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.bufferMutex.Lock()
			if len(s.buffer) > 0 {
				log.Println("Flushing buffer by timer")
				if err := s.flushBuffer(); err != nil {
					log.Printf("Failed to flush buffer: %v", err)
				}
			}
			s.bufferMutex.Unlock()
		case <-s.done:
			return
		}
	}
}

func (s *SinkServer) flushBuffer() error {
	if len(s.buffer) == 0 {
		return nil
	}

	_, err := s.fileWriter.Write(s.buffer)
	if err != nil {
		return err
	}

	if err := s.fileWriter.Flush(); err != nil {
		return err
	}

	s.buffer = s.buffer[:0]

	log.Printf("Flushed buffer to log file")
	return nil
}

func (s *SinkServer) Stop() {
	close(s.done)
}

func (s *SinkServer) Close() {
	s.bufferMutex.Lock()
	if len(s.buffer) > 0 {
		if err := s.flushBuffer(); err != nil { // Final flush
			log.Printf("Failed to flush buffer during shutdown: %v", err)
		}
	}
	s.bufferMutex.Unlock()

	if s.fileWriter != nil {
		s.fileWriter.Flush()
	}
	if s.logFile != nil {
		s.logFile.Close()
	}
}
