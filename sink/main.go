package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/sink/config"
	grpcserver "github.com/sink/server"
)

func main() {
	cfg, err := parseFlags()
	if err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	server, err := grpcserver.NewSinkServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create sink server: %v", err)
	}
	defer server.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping sink server...")
		server.Stop()
	}()

	log.Printf("Starting sink server on %s", cfg.BindAddr)
	log.Printf("Log file: %s", cfg.LogFilePath)
	log.Printf("Buffer size: %d bytes", cfg.BufferSize)
	log.Printf("Flush interval: %v", cfg.FlushInterval)
	log.Printf("Rate limit: %d bytes/sec", cfg.RateLimit)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func parseFlags() (config.Config, error) {
	var (
		cfg config.Config
		err error
	)

	flag.StringVar(&cfg.BindAddr, "bind-addr", ":9090", "Server bind address")
	flag.StringVar(&cfg.LogFilePath, "log-file", "telemetry.log", "Path to output log file")
	flag.IntVar(&cfg.BufferSize, "buffer-size", 1024*5, "Buffer size in bytes")
	flag.DurationVar(&cfg.FlushInterval, "flush-interval", 1*time.Minute, "Buffer flush interval")
	flag.IntVar(&cfg.RateLimit, "rate-limit", 1024*1024, "Rate limit in bytes per second")

	// TLS flags
	flag.BoolVar(&cfg.UseTLS, "tls", false, "Enable TLS")
	flag.StringVar(&cfg.CertFile, "cert-file", "", "Path to TLS certificate file")
	flag.StringVar(&cfg.KeyFile, "key-file", "", "Path to TLS private key file")
	flag.StringVar(&cfg.CAFile, "ca-file", "", "Path to CA certificate file (for mutual TLS)")

	// Encryption
	flag.BoolVar(&cfg.EnableEncryption, "encrypt", false, "Enable AES-GCM encryption for log data")
	flag.StringVar(&cfg.EncryptionKey, "encryption-key", "", "Base64 encoded 32-byte encryption key")

	if addr := os.Getenv("BIND_ADDR"); addr != "" {
		cfg.BindAddr = addr
	}
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		cfg.LogFilePath = logFile
	}
	if bufferSize := os.Getenv("BUFFER_SIZE"); bufferSize != "" {
		cfg.BufferSize, err = strconv.Atoi(bufferSize)
		if err != nil {
			return cfg, fmt.Errorf("invalid BUFFER_SIZE: %v", err)
		}
	}
	if flushInterval := os.Getenv("FLUSH_INTERVAL"); flushInterval != "" {
		if duration, err := time.ParseDuration(flushInterval); err == nil {
			cfg.FlushInterval = duration
		}
	}
	if rateLimit := os.Getenv("RATE_LIMIT"); rateLimit != "" {
		cfg.RateLimit, err = strconv.Atoi(rateLimit)
		if err != nil {
			return cfg, fmt.Errorf("invalid RATE_LIMIT: %v", err)
		}
	}

	flag.Parse()

	return cfg, nil
}
