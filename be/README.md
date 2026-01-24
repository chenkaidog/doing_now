# Backend Project

## Project Structure

```
be/
├── biz/
│   ├── config/       # Configuration loading and management
│   ├── dal/          # Data Access Layer (Repositories)
│   ├── db/           # Database initialization (MySQL, Redis)
│   ├── handler/      # HTTP Request Handlers (Controllers)
│   ├── middleware/   # HTTP Middlewares (JWT, Session, CORS, etc.)
│   ├── model/        # Data Models
│   │   ├── convert/  # Model conversion helpers
│   │   ├── domain/   # Domain models
│   │   ├── dto/      # Data Transfer Objects (API request/response)
│   │   ├── errs/     # Error definitions
│   │   └── storage/  # Database storage models
│   ├── router/       # Router registration
│   ├── service/      # Business Logic Layer
│   └── util/         # Utility functions
├── conf/             # Configuration files
├── docs/             # Swagger documentation
├── main.go           # Entry point
└── main_test.go      # Integration tests
```

## Code Conventions

- **Framework**: Hertz (HTTP framework)
- **Database**: GORM (ORM), go-redis (Redis client)
- **Architecture**: Layered architecture (Handler -> Service -> Repo/DB)
- **Authentication**: JWT (Access Token + Refresh Token) + Session
  - Access Token: Short-lived, passed via Authorization header.
  - Refresh Token: Long-lived, stored in HttpOnly Cookie, rotated on use.
- **Error Handling**: Custom error types in `biz/model/errs`, unified response format.
- **Testing**: Unit tests for services/utils, Integration tests in `main_test.go`.

## Common Commands

### Run Server
```bash
go run .
```
Or with specific environment/config if needed (default loads `conf/deploy.yml` or `conf/deploy.example.yml`).

### Run Tests
Run all tests including integration tests:
```bash
go test -gcflags="all=-N -l" ./...
```
Note: `-gcflags="all=-N -l"` is recommended when using `mockey` for mocking to disable inlining and optimizations.

### Run Specific Test
```bash
go test -v -gcflags="all=-N -l" -run TestName ./path/to/package
```

### Generate Swagger Docs
If using swag CLI:
```bash
swag init -g main.go -o docs
```

## Environment Setup
Ensure `conf/deploy.yml` exists (copy from `conf/deploy.example.yml`) and configure MySQL/Redis connections.
