# EchoAuth

A robust, secure authentication service built with Go, featuring account management, password reset functionality, rate limiting, and comprehensive monitoring.

## Features

- **User Authentication**
  - Secure login with JWT tokens
  - Password hashing using bcrypt
  - Account lockout after multiple failed attempts
  - Rate limiting for login attempts
  - Session management with Redis

- **User Management**
  - User registration with email verification
  - Password reset functionality
  - Email notifications for account actions
  - Profile management
  - Account deactivation

- **Security Features**
  - Strong password validation
  - Email validation
  - Protection against brute force attacks
  - Rate limiting across different endpoints
  - Account lockout mechanism
  - SSL/TLS encryption for database connections
  - Secure session management
  - JWT token blacklisting

- **Monitoring & Observability**
  - Prometheus metrics integration
  - Grafana dashboards
  - Structured logging with zerolog
  - Health check endpoints
  - Request duration tracking
  - Database and Redis metrics
  - Real-time monitoring alerts

## Architecture

```
├── cmd/            # Application entrypoint
├── config/         # Configuration management
├── controllers/    # HTTP request handlers
├── database/       # Database connection and migrations
├── middlewares/    # HTTP middleware components
├── models/         # Data models
├── repositories/   # Data access layer
├── services/       # Business logic
├── utils/         # Utility functions
├── migrations/    # Database migrations
├── scripts/       # Deployment and maintenance scripts
└── grafana/       # Grafana dashboards and configuration
```

## Prerequisites

- Go 1.22 or higher
- PostgreSQL 14 or higher
- Redis 7 or higher
- Docker and Docker Compose
- OpenSSL (for SSL certificate generation)

## Development

### Quick Start with Docker

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/EchoAuth.git
   cd EchoAuth
   ```

2. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

3. Start the development environment:
   ```bash
   docker compose -f docker-compose.dev.yml up -d
   ```

Development environment features:
- Hot-reload using Air
- Mounted source code for live editing
- Cached Go modules
- Isolated development databases
- Automatic database migrations
- Development SSL certificates

### Development Services

| Service     | URL                      | Default Credentials |
|-------------|--------------------------|-------------------|
| Auth API    | http://localhost:8080    | N/A              |
| Grafana     | http://localhost:3000    | admin/admin      |
| Prometheus  | http://localhost:9090    | N/A              |
| MailHog     | http://localhost:8025    | N/A              |

### Testing

The project includes a dedicated test environment:

```bash
# Run all tests
docker compose -f docker-compose.test.yml run --rm test

# Run specific tests
docker compose -f docker-compose.test.yml run --rm test go test ./path/to/package

# Generate coverage report
docker compose -f docker-compose.test.yml run --rm test go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Production Deployment

### SSL Certificate Setup

1. Generate production certificates:
   ```bash
   ./scripts/generate-prod-certs.sh
   ```

2. Configure certificate renewal:
   ```bash
   # Add to crontab (30 days before expiry)
   0 0 1 * * /path/to/scripts/renew-server-cert.sh
   ```

### Environment Configuration

1. Create production environment file:
   ```bash
   cp .env.example .env
   ```

2. Update production values:
   - Generate strong passwords and secrets
   - Configure proper SMTP settings
   - Set production database credentials
   - Configure SSL certificates

### Deployment

1. Build and start services:
   ```bash
   docker compose up -d
   ```

2. Verify deployment:
   ```bash
   # Check service health
   curl http://localhost:8080/health

   # Verify SSL configuration
   ./scripts/verify-ssl.sh your-database-host
   ```

### Monitoring Setup

1. Access Grafana (http://localhost:3000)
2. Import provided dashboards from `grafana/dashboards/`
3. Configure alerting in Grafana
4. Set up monitoring alerts:
   - Certificate expiration
   - Failed login attempts
   - High rate limiting hits
   - Error rate thresholds
   - Resource usage alerts

## API Documentation

### Security Requirements

#### Password Policy
- Minimum 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- At least one special character
- Cannot be common passwords (e.g., password123, 12345678)

#### Rate Limiting
- Login attempts: 5 per minute per IP
- Password reset requests: 3 per hour per email
- API endpoints: 100 requests per minute per IP

#### Account Security
- Account lockout after 5 failed login attempts
- Lockout duration: 15 minutes
- Password reset tokens expire after 1 hour

### Authentication Endpoints

- POST `/api/EchoAuth/register`
  ```json
  {
    "email": "user@example.com",
    "password": "SecurePass123!",
    "first_name": "John",
    "last_name": "Doe"
  }
  ```
  Response: `201 Created`
  ```json
  {
    "message": "User registered successfully"
  }
  ```
  Error Responses:
  - `400 Bad Request`: Invalid input (email format, password requirements)
  - `409 Conflict`: Email already registered

- POST `/api/EchoAuth/login`
  ```json
  {
    "email": "user@example.com",
    "password": "SecurePass123!"
  }
  ```
  Response: `200 OK`
  ```json
  {
    "access_token": "eyJhbG...",
    "refresh_token": "eyJhbG...",
    "token_type": "Bearer",
    "expires_in": 86400,
    "user": {
      "id": 1,
      "email": "user@example.com",
      "first_name": "John",
      "last_name": "Doe",
      "created_at": "2024-01-14T...",
      "updated_at": "2024-01-14T..."
    }
  }
  ```
  Error Responses:
  - `400 Bad Request`: Invalid credentials
  - `429 Too Many Requests`: Rate limit exceeded
  - `403 Forbidden`: Account locked

- POST `/api/EchoAuth/refresh`
  ```json
  {
    "refresh_token": "eyJhbG..."
  }
  ```
  Response: `200 OK`
  ```json
  {
    "access_token": "eyJhbG...",
    "refresh_token": "eyJhbG...",
    "token_type": "Bearer",
    "expires_in": 86400
  }
  ```
  Error Responses:
  - `400 Bad Request`: Invalid refresh token
  - `401 Unauthorized`: Expired refresh token

- POST `/api/EchoAuth/logout` (Protected)
  ```json
  {
    "refresh_token": "eyJhbG..."
  }
  ```
  Headers:
  ```
  Authorization: Bearer <access_token>
  ```
  Response: `200 OK`
  Error Responses:
  - `401 Unauthorized`: Invalid access token

- POST `/api/EchoAuth/forgot-password`
  ```json
  {
    "email": "user@example.com"
  }
  ```
  Response: `200 OK`
  ```json
  {
    "message": "If your email is registered, you will receive a reset link shortly"
  }
  ```
  Error Responses:
  - `429 Too Many Requests`: Rate limit exceeded

- POST `/api/EchoAuth/reset-password`
  ```json
  {
    "token": "reset-token",
    "new_password": "NewSecurePass123!"
  }
  ```
  Response: `200 OK`
  ```json
  {
    "message": "Password reset successfully"
  }
  ```
  Error Responses:
  - `400 Bad Request`: Invalid token or password requirements not met
  - `401 Unauthorized`: Expired reset token

### Health Check

- GET `/health`
  Response: `200 OK`
  ```json
  {
    "status": "healthy",
    "database": "up",
    "redis": "up",
    "version": "1.0.0"
  }
  ```
  Error Response:
  - `503 Service Unavailable`: Service unhealthy

### Metrics

- GET `/metrics`
  Response: `200 OK`
  Prometheus metrics format

### Security Headers

All endpoints include the following security headers:
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains (production only)
```

### Rate Limiting

All endpoints are protected by rate limiting:
- 5 attempts per minute per IP for login
- 100 requests per minute for other endpoints
- Rate limit headers included in response:
  ```
  X-RateLimit-Limit: 100
  X-RateLimit-Remaining: 99
  X-RateLimit-Reset: 60
  ```

## Configuration

The service is configured via environment variables:

```bash
# Server
PORT=8080
ENV=development

# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=auth_db

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# Rate Limiting
RATE_LIMIT=100
RATE_LIMIT_WINDOW=60

# Email (Development)
SMTP_HOST=mailhog
SMTP_PORT=1025
```

## Security Considerations

1. Password Requirements:
   - Minimum 8 characters
   - At least one uppercase letter
   - At least one lowercase letter
   - At least one number
   - At least one special character
   - Not in common password list

2. Rate Limiting:
   - 5 attempts per minute per IP
   - Account lockout after 5 failed attempts
   - 15-minute lockout duration

3. SSL/TLS Security:
   - TLS 1.2 minimum
   - Strong cipher suites
   - Regular certificate rotation
   - Secure key storage

4. Database Security:
   - Encrypted connections
   - Strong password policies
   - Regular security updates
   - Backup encryption

## Monitoring

The service exposes Prometheus metrics at `/metrics`, including:
- Authentication attempts
- Active sessions
- Rate limit hits
- Request durations
- Database operations
- Redis and PostgreSQL metrics

## Maintenance

### Regular Tasks

1. Certificate Management:
   ```bash
   # Check certificate expiration
   ./scripts/check-cert-expiry.sh

   # Renew certificates
   ./scripts/renew-server-cert.sh
   ```

2. Database Maintenance:
   ```bash
   # Run migrations
   docker compose exec auth go run cmd/migrate/main.go up

   # Backup database
   docker compose exec postgres pg_dump -U $POSTGRES_USER $POSTGRES_DB > backup.sql
   ```

3. Monitoring:
   - Review Grafana dashboards
   - Check error logs
   - Monitor resource usage
   - Review security alerts

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests (`docker compose -f docker-compose.test.yml run --rm test`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support, please open an issue in the GitHub repository or contact the maintainers.
 