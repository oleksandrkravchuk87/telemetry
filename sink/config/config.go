package config

import "time"

type Config struct {
	BindAddr      string
	LogFilePath   string
	BufferSize    int
	FlushInterval time.Duration
	RateLimit     int // bytes per second

	// TLS configuration
	UseTLS   bool
	CertFile string
	KeyFile  string
	CAFile   string

	// Encryption
	EnableEncryption bool
	EncryptionKey    string
}
