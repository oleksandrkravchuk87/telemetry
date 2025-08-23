#!/bin/bash

mkdir -p ../certs
cd ../certs

# Generate CA private key
openssl genrsa -out ca-key.pem 4096

# Generate CA certificate
openssl req -new -x509 -key ca-key.pem -sha256 -subj "/C=UA/ST=Lviv/L=Lviv/O=Test/OU=Dev/CN=Telemetry-CA" -days 365 -out ca-cert.pem

# Generate server private key
openssl genrsa -out server-key.pem 4096

# Create server configuration file with proper sections
cat > server.conf <<EOF
[req]
default_bits = 4096
prompt = no
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
C = UA
ST = Lviv
L = Lviv
O = Test
OU = Dev
CN = localhost

[v3_req]
basicConstraints = CA:FALSE
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
IP.3 = 0.0.0.0
EOF

# Generate server certificate signing request
openssl req -new -key server-key.pem -out server.csr -config server.conf

# Generate server certificate signed by CA
openssl x509 -req -in server.csr -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem -days 365 -sha256 -extensions v3_req -extfile server.conf

# Generate client private key
openssl genrsa -out client-key.pem 4096

# Create client configuration file
cat > client.conf <<EOF
[req]
default_bits = 4096
prompt = no
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
C = UA
ST = Lviv
L = Lviv
O = Test
OU = Dev
CN = sensor-client

[v3_req]
basicConstraints = CA:FALSE
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
EOF

# Generate client certificate signing request
openssl req -new -key client-key.pem -out client.csr -config client.conf

# Generate client certificate signed by CA
openssl x509 -req -in client.csr -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out client-cert.pem -days 365 -sha256 -extensions v3_req -extfile client.conf

# Clean up temporary files
rm server.csr client.csr server.conf client.conf

echo "Certificates generated in ../certs/:"
echo "  CA: ca-cert.pem, ca-key.pem"
echo "  Server: server-cert.pem, server-key.pem"
echo "  Client: client-cert.pem, client-key.pem"

# Verify certificates
echo ""
echo "Verifying certificates..."
openssl verify -CAfile ca-cert.pem server-cert.pem
openssl verify -CAfile ca-cert.pem client-cert.pem

# Show certificate details
echo ""
echo "Server certificate details:"
openssl x509 -in server-cert.pem -text -noout | grep -A 1 "Subject:"
openssl x509 -in server-cert.pem -text -noout | grep -A 3 "Subject Alternative Name"

echo ""
echo "Client certificate details:"
openssl x509 -in client-cert.pem -text -noout | grep -A 1 "Subject:"