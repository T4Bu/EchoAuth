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

## Configuration

The service uses environment variables for configuration. Create a `.env` file in the root directory:

```env
PORT=8080
DATABASE_URL=host=localhost user=postgres password=postgres dbname=auth_db port=5432 sslmode=disable
JWT_SECRET=your-super-secret-key-change-this-in-production

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
# REDIS_DB=0 (using default)

# SMTP Configuration (for email notifications)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-specific-password
SMTP_FROM=noreply@yourdomain.com

# Logging Configuration
ENV=development
LOG_LEVEL=debug
```

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd auth
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set up the database:
   ```bash
   # Using Docker
   docker-compose up -d postgres redis
   
   # Or manually create a PostgreSQL database named 'auth_db'
   ```

4. Run migrations:
   ```bash
   # The service will automatically run migrations on startup
   ```

5. Start the service:
   ```bash
   go run cmd/main.go
   ```

## API Endpoints

### Authentication

#### Register User
```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "first_name": "John",
  "last_name": "Doe"
}
```

#### Login
```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!"
}
```

#### Logout
```http
POST /auth/logout
Authorization: Bearer <token>
```

### Password Reset

#### Request Reset Token
```http
POST /auth/reset-password/request
Content-Type: application/json

{
  "email": "user@example.com"
}
```

#### Reset Password
```http
POST /auth/reset-password/reset
Content-Type: application/json

{
  "token": "reset-token",
  "new_password": "NewSecurePass123!"
}
```

### Health Check

```http
GET /health
```

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
docker-compose -f docker-compose.test.yml up -d
INTEGRATION_TEST=true go test ./...
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

## Metrics

The service exposes Prometheus metrics at `/metrics`, including:
- Authentication attempts
- Active sessions
- Rate limit hits
- Request durations
- Database operations

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Docker Support

### Quick Start with Docker

1. Copy the example environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit the `.env` file with your configuration.

3. Build and start all services:
   ```bash
   docker-compose up -d
   ```

4. Check service health:
   ```bash
   docker-compose ps
   ```

### Available Services

- **Auth Service**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000
  - Default credentials: admin/admin

### Docker Commands

```bash
# Build the auth service
docker-compose build auth

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f auth

# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v

# Scale the auth service (if needed)
docker-compose up -d --scale auth=3
```

### Production Deployment

For production deployment, consider the following:

1. Use a proper secrets management solution
2. Configure SSL/TLS termination
3. Use a container orchestration platform (e.g., Kubernetes)
4. Set up proper monitoring and alerting
5. Use separate databases for different environments

### Container Health Checks

The following health checks are configured:

- **Auth Service**: HTTP check on `/health` endpoint
- **PostgreSQL**: `pg_isready` command
- **Redis**: `redis-cli ping` command

### Data Persistence

Docker volumes are used to persist data:

- `postgres_data`: PostgreSQL data
- `redis_data`: Redis data
- `prometheus_data`: Prometheus metrics
- `grafana_data`: Grafana dashboards and settings 