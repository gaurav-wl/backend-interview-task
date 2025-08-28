# Explore Service

A high-performance gRPC microservice for managing user interactions in a dating/matching application. Built with Go and PostgreSQL and Redis, containerized with Docker.

## Overview

This service manages user decisions (likes/passes) and provides endpoints to:
- Record user decisions (like/pass)
- List users who liked a specific user
- List new likes (users who liked but haven't been liked back)
- Count total likes received by a user
- Detect mutual likes

### Components
- **gRPC Service**: handles all client interactions, requests validation, and response formatting
- **Repository Layer**: Data access layer with PostgreSQL
- **Core Layer**: Handles the business logic
- **Providers**: All external dependencies (DB, cache, etc.)
- **Configuration**: Managed with Viper, supports config files and environment variables

### Tools and Libraries Used:
- **sqlc**: for type-safe SQL queries
  - [sqlc documentation](https://sqlc.dev/)
- **mockery**: for generating mocks for unit tests
- **viper**: for configuration management

## Quick Start

### Prerequisites

- Go 1.24 or later
- Docker and Docker Compose
- Protocol Buffers compiler (protoc)

### Local Development

1. **Install dependencies:**
   ```bash
   make deps
   ```

2. **Generate protobuf, mocks, sqlc files:**
   ```bash
   make generate-all
   ```

3. **Start other services:**
   ```bash
   docker compose -f docker-compose-local.yml up -d
   ```

4. **Run the service:**
   ```bash
   make run
   ```

### Docker Development

1. **Start everything with Docker:**
   ```bash
   make docker-up
   ```

### Test the service methods
   ```bash
   # Install grpcui for testing
   go install github.com/fullstorydev/grpcui/cmd/grpcui@latest

   # Test the service
   grpcui -plaintext localhost:8080
   ```

## Testing

### Unit Tests
```bash
make test-unit
```

### Adding New Features

1. Update protobuf definitions in `proto/`
2. Regenerate code with `make proto`
3. If wanted to add new DB queries, update `db/query` and run `make sqlc`
4. To generate mocks, run `make mock`
5. Implement repository methods
6. Implement service methods
