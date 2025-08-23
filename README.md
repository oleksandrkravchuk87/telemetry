# Telemetry System

A distributed telemetry system consisting of sensor nodes that generate data and a sink server that collects and logs the data using gRPC communication.

## Architecture

- **Sensor Node**: Generates telemetry data at configurable rates and sends to sink server with retry logic
- **Sink Server**: Receives telemetry data, applies rate limiting, buffers messages, and writes to log files

## Prerequisites

- Go 1.19 or later
- Protocol Buffers compiler (`protoc`)

## Installation

1. Clone the repository:
````` 
git clone <repository-url>
cd telemetry
````` 
2. Install dependencies:

# For sensor node
````` 
cd sensor_node
make dependencies
make generate
````` 
# For sink server
````` 
cd sink
make dependencies
make generate
````` 

### Sensor Node
````` 
cd sensor_node
make build-linux     # Linux binary
make build-windows   # Windows binary
make build-macos     # macOS Intel binary
make build-macos-arm # macOS ARM binary
make build-all       # All platforms
````` 
### Sink Server
````` 
cd sink
make build
````` 
### TLS Certificate Setup
Generate self-signed certificates for development:
#### TLS:
````` ./scripts/generate-certs.sh`````
#### mTLS:
````` ./scripts/generate-mtls-certs.sh`````

## Running the Applications

### 1. Start the Sink Server
````` 
cd sink
./bin/server [options]
````` 
**Command line options:**
- `--bind-addr`: Server bind address (default: `:9090`)
- `--log-file`: Path to output log file (default: `telemetry.log`)
- `--buffer-size`: Buffer size in bytes (default: `5120`)
- `--flush-interval`: Buffer flush interval (default: `1m`)
- `--rate-limit`: Rate limit in bytes per second (default: `1048576`)
- `--tls`: Enable TLS (default: false)
- `--cert-file`: Path to TLS certificate file
- `--key-file`: Path to TLS private key file
- `--ca-file`: Path to CA certificate file (for mutual TLS)
- `--encrypt`: Enable AES-GCM encryption for log data (default: false)
- `--encryption-key`: Base64 encoded 32-byte encryption key

**Environment variables:**
- `BIND_ADDR`: Override bind address
- `LOG_FILE`: Override log file path
- `BUFFER_SIZE`: Override buffer size
- `FLUSH_INTERVAL`: Override flush interval
- `RATE_LIMIT`: Override rate limit

**Example:**
Basic server:
````` 
./bin/server --bind-addr=":9090" --log-file="telemetry.log" --buffer-size=8192 --flush-interval="5s" --rate-limit=2097152
`````

Server with TLS:
````` 
./bin/server --tls --cert-file=../certs/server-cert.pem --key-file=../certs/server-key.pem
````` 
Server with mTLS:
````` 
./bin/server --tls --cert-file=../certs/server-cert.pem --key-file=../certs/server-key.pem --ca-file=../certs/ca-cert.pem
````` 
Server with custom encryption key:
````` 
ENCRYPTION_KEY=$(openssl rand -base64 32)
./bin/server --encrypt --encryption-key="$ENCRYPTION_KEY"
````` 
### 2. Start Sensor Nodes
````` 
cd sensor_node
./bin/sensor_node-linux-amd64 [options]
````` 
**Command line options:**
- `--rate`: Number of messages per second (default: `1.0`)
- `--sensor-name`: Name of the sensor (default: `"default-sensor"`)
- `--sink-addr`: Address of the telemetry sink (default: `"localhost:9090"`)
- `--tls`: Use TLS for connection (default: false)
- `--cert-file`: Path to TLS certificate file (optional)
- `--client-cert`: Path to client certificate file (for mTLS)
- `--client-key`: Path to client private key file (for mTLS)

**Example:**

## Single sensor:
````` 
./bin/sensor_node-linux-amd64 --sensor-name="temperature-01" --rate=2.0 --sink-addr="localhost:9090"
````` 
## Single sensor with TLS:
````` 
./bin/sensor_node-linux-amd64 --sensor-name="temperature-01" --rate=2.0 --tls --cert-file=../certs/ca-cert.pem
````` 
## Single sensor with mutual TLS:
````` 
./bin/sensor_node-linux-amd64 --sensor-name="temperature-01" --rate=2.0 --tls --cert-file=../certs/ca-cert.pem --client-cert=../certs/client-cert.pem --client-key=../certs/client-key.pem
````` 
## Multiple sensors:
````` 
#### Terminal 1
./bin/sensor_node-linux-amd64 --sensor-name="temperature-01" --rate=1.0

#### Terminal 2
./bin/sensor_node-linux-amd64 --sensor-name="humidity-01" --rate=0.5

#### Terminal 3
./bin/sensor_node-linux-amd64 --sensor-name="pressure-01" --rate=2.0
````` 
# Client in Docker container:
````` 
sudo docker build -t telemetry/sensor-node .
docker run --rm --network host   telemetry/sensor-node   --sensor-name="docker-temp-01"   --rate=2.0   --sink-addr="localhost:9090"
````` 
## Cleanup

# Clean build artifacts
````` 
cd sensor_node && make clean
cd ../sink && make clean
````` 
# Remove log files
````` 
rm -f sink/telemetry.log
````` 
