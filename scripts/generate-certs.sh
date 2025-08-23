#!/bin/bash

mkdir -p ../certs

openssl genrsa -out ../certs/ca-key.pem 4096

openssl req -new -x509 -key ../certs/ca-key.pem -sha256 -subj "/C=UA/ST=Lviv/L=Lviv/O=Test/OU=Dev/CN=localhost" -days 3650 -out ../certs/ca-cert.pem

openssl genrsa -out ../certs/server-key.pem 4096

cat > ../certs/server.conf <<EOF
[req]
default_bits = 4096
prompt = no
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
C = UA
ST = Lviv
O = Test
CN = localhost

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

openssl req -new -key ../certs/server-key.pem -out ../certs/server.csr -config ../certs/server.conf

# Generate server certificate signed by CA with SAN
openssl x509 -req -in ../certs/server.csr -CA ../certs/ca-cert.pem -CAkey ../certs/ca-key.pem -CAcreateserial -out ../certs/server-cert.pem -days 365 -sha256 -extensions v3_req -extfile ../certs/server.conf
rm ../certs/server.csr ../certs/server.conf

echo "Certificates generated in certs/ directory"