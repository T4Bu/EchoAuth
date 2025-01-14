#!/bin/bash
set -e

# Configuration (should match your original settings)
CERT_DIR="certs"
DAYS_VALID_CERT=365  # 1 year for server cert
COUNTRY="US"
STATE="State"
LOCALITY="City"
ORGANIZATION="Your Organization"
ORG_UNIT="IT"
SERVER_CN="db.yourdomain.com"  # Change this to your database hostname
KEY_SIZE=4096

# Check if CA exists
if [ ! -f "${CERT_DIR}/ca/ca.key" ] || [ ! -f "${CERT_DIR}/ca/ca.crt" ]; then
    echo "Error: CA files not found in ${CERT_DIR}/ca/"
    echo "Please run generate-prod-certs.sh first to create a CA"
    exit 1
fi

cd "${CERT_DIR}"

# Backup existing certificates
echo "Backing up existing certificates..."
BACKUP_DIR="backup/$(date +%Y%m%d_%H%M%S)"
mkdir -p "${BACKUP_DIR}"
[ -f deploy/postgresql.key ] && cp deploy/postgresql.key "${BACKUP_DIR}/"
[ -f deploy/postgresql.crt ] && cp deploy/postgresql.crt "${BACKUP_DIR}/"

# Generate new server private key
echo "Generating new server private key..."
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

# Update deployment bundle
echo "Updating deployment bundle..."
cp server/server.key deploy/postgresql.key
cp server/server.crt deploy/postgresql.crt
chmod 600 deploy/postgresql.key
chmod 644 deploy/postgresql.crt

echo "
Certificate renewal complete!

New files generated:
- deploy/postgresql.key  : New server private key
- deploy/postgresql.crt  : New server certificate

Previous certificates backed up to: ${BACKUP_DIR}

To deploy:
1. Copy the new files from the 'deploy' directory to your PostgreSQL data directory:
   cp deploy/postgresql.* /path/to/postgres/data/

2. Restart PostgreSQL to apply the new certificate:
   systemctl restart postgresql
   # or
   pg_ctl restart

3. Verify the new certificate:
   ./verify-ssl.sh your-database-host

Note: The root.crt (CA certificate) remains unchanged and clients don't need to update it.
" 