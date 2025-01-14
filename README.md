# Authentication Service

A robust, secure authentication service built with Go, featuring account management, password reset functionality, and rate limiting.

## Features

- **User Authentication**
  - Secure login with JWT tokens
  - Password hashing using bcrypt
  - Account lockout after multiple failed attempts
  - Rate limiting for login attempts

- **User Management**
  - User registration with email verification
  - Password reset functionality
  - Email notifications for account actions
  - Session management

- **Security Features**
  - Strong password validation
  - Email validation
  - Protection against brute force attacks
  - Rate limiting across different endpoints
  - Account lockout mechanism

- **Monitoring & Logging**
  - Prometheus metrics integration
  - Structured logging with zerolog
  - Health check endpoints
  - Request duration tracking

## Prerequisites

- Go 1.22 or higher
- PostgreSQL 14 or higher
- Redis 6 or higher
- Docker (optional, for containerization)

## Development

### Quick Start with Docker

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Start the development environment:
   ```bash
   docker compose -f docker-compose.dev.yml up -d
   ```

Features:
- Hot-reload using Air (automatic rebuilds on code changes)
- Mounted source code for live editing
- Cached Go modules
- Isolated development databases

### Development Services

- Auth Service: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090

### Useful Commands

```bash
# View logs
docker compose -f docker-compose.dev.yml logs -f auth

# Access container shell
docker compose -f docker-compose.dev.yml exec auth sh

# Run tests
docker compose -f docker-compose.test.yml up -d
go test ./...

# Stop environment
docker compose -f docker-compose.dev.yml down
```

## API Documentation

### Authentication Endpoints

- POST `/EchoAuth/register`
  ```json
  {
    "email": "user@example.com",
    "password": "SecurePass123!",
    "first_name": "John",
    "last_name": "Doe"
  }
  ```

- POST `/EchoAuth/login`
  ```json
  {
    "email": "user@example.com",
    "password": "SecurePass123!"
  }
  ```

- POST `/EchoAuth/logout`
  - Requires JWT token in Authorization header

### Password Reset

- POST `/EchoAuth/reset-password/request`
  ```json
  {
    "email": "user@example.com"
  }
  ```

- POST `/EchoAuth/reset-password/reset`
  ```json
  {
    "token": "reset-token",
    "new_password": "NewSecurePass123!"
  }
  ```

### Health Check

```http
GET /health
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

3. JWT Tokens:
   - 24-hour expiration
   - Secure token validation
   - Token blacklisting on logout

## Monitoring

The service exposes Prometheus metrics at `/metrics`, including:
- Authentication attempts
- Active sessions
- Rate limit hits
- Request durations
- Database operations
- Redis and PostgreSQL metrics

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
 