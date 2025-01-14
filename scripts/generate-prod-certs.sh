#!/bin/bash
set -e

# Configuration
CERT_DIR="certs"
DAYS_VALID=3650  # 10 years for CA, adjust as needed
DAYS_VALID_CERT=365  # 1 year for server cert
COUNTRY="US"
STATE="State"
LOCALITY="City"
ORGANIZATION="Your Organization"
ORG_UNIT="IT"
CA_CN="Your Organization Root CA"
SERVER_CN="db.yourdomain.com"  # Change this to your database hostname
KEY_SIZE=4096

# Create directories
mkdir -p "${CERT_DIR}"/{ca,server}
cd "${CERT_DIR}"

# Generate CA private key and certificate
echo "Generating CA private key and certificate..."
openssl genrsa -out ca/ca.key "${KEY_SIZE}"
chmod 400 ca/ca.key

openssl req -new -x509 -days "${DAYS_VALID}" \
    -key ca/ca.key \
    -out ca/ca.crt \
    -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORGANIZATION}/OU=${ORG_UNIT}/CN=${CA_CN}"

# Generate server private key
echo "Generating server private key..."
openssl genrsa -out server/server.key "${KEY_SIZE}"
chmod 400 server/server.key

# Generate server CSR
echo "Generating server certificate signing request..."
openssl req -new \
    -key server/server.key \
    -out server/server.csr \
    -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORGANIZATION}/OU=${ORG_UNIT}/CN=${SERVER_CN}"

# Create server certificate extensions file
cat > server/server.ext << EOF
basicConstraints = CA:FALSE
nsCertType = server
nsComment = "PostgreSQL Server Certificate"
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid,issuer:always
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${SERVER_CN}
DNS.2 = localhost
IP.1 = 127.0.0.1
EOF

# Sign the server certificate with our CA
echo "Signing server certificate with CA..."
openssl x509 -req \
    -days "${DAYS_VALID_CERT}" \
    -in server/server.csr \
    -CA ca/ca.crt \
    -CAkey ca/ca.key \
    -CAcreateserial \
    -out server/server.crt \
    -extfile server/server.ext

# Verify the certificate
echo "Verifying certificate..."
openssl verify -CAfile ca/ca.crt server/server.crt

# Create deployment bundle
echo "Creating deployment bundle..."
mkdir -p deploy
cp server/server.key deploy/postgresql.key
cp server/server.crt deploy/postgresql.crt
cp ca/ca.crt deploy/root.crt
chmod 600 deploy/postgresql.key
chmod 644 deploy/postgresql.crt deploy/root.crt

# Create verification script
cat > verify-ssl.sh << 'EOF'
#!/bin/bash
echo | openssl s_client -connect "$1:5432" -starttls postgres 2>/dev/null | openssl x509 -text
EOF
chmod +x verify-ssl.sh

echo "
Certificate generation complete!

Files generated:
- deploy/postgresql.key  : Server private key
- deploy/postgresql.crt  : Server certificate
- deploy/root.crt       : CA certificate

To deploy:
1. Copy the files from the 'deploy' directory to your PostgreSQL data directory
2. Update postgresql.conf with:
   ssl = on
   ssl_cert_file = 'postgresql.crt'
   ssl_key_file = 'postgresql.key'
   ssl_ca_file = 'root.crt'

3. Distribute root.crt to clients for verification

To verify SSL connection:
./verify-ssl.sh your-database-host

Note: Keep the CA key (ca/ca.key) secure for future certificate generation.
" 