#!/bin/sh
set -e

# Create SSL directory
mkdir -p /var/lib/postgresql/data/ssl
cd /var/lib/postgresql/data/ssl

# Generate self-signed certificate and key
openssl req -new -x509 -days 365 -nodes \
  -out server.crt \
  -keyout server.key \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Set correct permissions
chmod 600 server.key
chown postgres:postgres server.key server.crt

# Update PostgreSQL configuration
cat >> /var/lib/postgresql/data/postgresql.conf << EOF
# SSL configuration
ssl = on
ssl_cert_file = '/var/lib/postgresql/data/ssl/server.crt'
ssl_key_file = '/var/lib/postgresql/data/ssl/server.key'
ssl_prefer_server_ciphers = on
ssl_min_protocol_version = 'TLSv1.2'
EOF

# Configure client authentication
cat >> /var/lib/postgresql/data/pg_hba.conf << EOF
# Allow password authentication with SSL
hostssl all             all             0.0.0.0/0               scram-sha-256
hostssl all             all             ::/0                    scram-sha-256
EOF 