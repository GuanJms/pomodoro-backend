# Introduction and Overview

The pomodoro backend system provides functionality that runs the single pomodoro at server end with persistence.

The system provides account managmenet that includes autherization, registration, password management, and email verification.

## Requirements

# System Architecture

![System Design Diagram](./docs/architecture/pomodoro-project-backend.svg)

## Rationale

# Data Design

![Data Design Diagram](./docs/data_design/data_design.svg)

# User Interface Design

![User Interface Diagram](./docs/use_case/use_case.svg)

# API

## API Reference

See full OpenAPI spec: [openapi.yaml](./api/openapi.yaml)

---

## Setup & Installation

### Prerequisites

- Docker and Docker Compose
- Go 1.24+ (for development)

### Database Setup

The application uses PostgreSQL with automatic schema initialization:

1. **Copy environment template:**

   ```bash
   cp .env.example .env
   ```

2. **Choose Configuration Method:**

   **Option A: Environment Variables (Recommended for Production)**
   Edit `.env` file with your preferred values:

   ```bash
   POSTGRES_USER=pomodoro_user
   POSTGRES_PASSWORD=your_secure_password_here
   POSTGRES_DB=pomodoro_db
   DSN=postgres://pomodoro_user:your_secure_password_here@localhost:5432/pomodoro_db?sslmode=disable
   JWT_SECRET=your-super-secure-jwt-secret-key-change-in-production
   WEB_PORT=8080
   REDIS_ADDR=localhost:6379
   ```

   **Option B: Hardcoded Values (For Development/Quick Start)**
   The application will use hardcoded values by default. No `.env` file needed.

   **How to Switch Between Methods:**
   Edit `cmd/main.go` and comment/uncomment the appropriate lines:

   ```go
   // For Environment Variables:
   // var dsn = os.Getenv("DSN")
   var dsn = "postgres://..." // comment out this line

   // For Hardcoded Values:
   var dsn = "postgres://..." // keep this line
   // var dsn = os.Getenv("DSN") // comment out this line
   ```

3. **Start services:**
   ```bash
   docker-compose up -d
   ```

### Database Initialization

When you start the PostgreSQL container for the first time, it will automatically:

1. Create the database using the environment variables
2. Execute `scripts/postgres/schema.sql` to create tables and indexes
3. Set up the user role enum and initial schema

The schema includes:

- **UUID extension** for user ID generation
- **User roles enum** (USER, ADMIN)
- **Users table** with proper constraints
- **Performance indexes** on username, email, and role columns

### Database Management

**Reset Database (Development):**

```bash
# Stop and remove containers
docker-compose down

# Remove database data
sudo rm -rf db-data/

# Restart with fresh database
docker-compose up -d
```

**View Database Logs:**

```bash
docker-compose logs postgres
```

**Connect to Database:**

```bash
docker-compose exec postgres psql -U $POSTGRES_USER -d $POSTGRES_DB
```

### Development

1. **Install dependencies:**

   ```bash
   go mod tidy
   ```

2. **Generate SQL code:**

   ```bash
   sqlc generate
   ```

3. **Run the application:**
   ```bash
   go run cmd/main.go
   ```

## Data Models

### Session Type Values

- `WORK`: 25 min work session
- `SHORT_BREAK` : 5 min short break
- `LONG_BREAK` : 20 min long break

---

## Authentication & Authorization

The API uses JWT Bearer token authentication with role-based access control (RBAC) for protected endpoints.

### JWT Authentication

Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

JWT tokens expire after **24 hours** and contain user identification information:

- User ID
- Username

**Note:** User roles are not stored in JWT tokens. Instead, the current role is fetched from the database on each request to ensure role changes take effect immediately without requiring token refresh.

### Role-Based Access Control (RBAC)

The system implements two user roles with different privilege levels:

#### User Roles

- **USER**: Basic user role (default for new registrations)
- **ADMIN**: Administrative role with elevated privileges

#### Permission Matrix

| Endpoint              | USER Role | ADMIN Role | Description                           |
| --------------------- | --------- | ---------- | ------------------------------------- |
| `POST /auth/register` | Public    | Public     | User registration (creates USER role) |
| `POST /auth/login`    | Public    | Public     | User authentication                   |
| `GET /auth/profile`   | ✅        | ✅         | View user profile                     |
| `GET /system/state`   | ✅        | ✅         | View system state                     |
| `POST /system/start`  | ❌        | ✅         | Start/modify pomodoro system          |

### API Endpoints

#### Authentication Endpoints

- `POST /auth/register` - Register new user (creates USER role)
- `POST /auth/register-admin` - Register admin user (creates ADMIN role) - **DEVELOPMENT ONLY**
- `POST /auth/login` - User login with JWT token response
- `GET /auth/profile` - Get user profile (requires authentication)

#### System Endpoints

- `GET /system/state` - Get current pomodoro system state (requires USER+ role)
- `POST /system/start` - Start new pomodoro session (requires ADMIN role)

### Testing Role-Based Access Control

#### 1. Register Users

```bash
# Register regular user (USER role)
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"user1","password":"password","email":"user1@example.com"}'

# Register admin user (ADMIN role) - Development only!
curl -X POST http://localhost:8080/auth/register-admin \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"adminpass","email":"admin@example.com"}'
```

#### 2. Login and Get Tokens

```bash
# Login as regular user
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user1","password":"password"}'

# Login as admin user
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"adminpass"}'
```

#### 3. Test Permissions

**USER Role Permissions:**

```bash
# ✅ Can view system state
curl -H "Authorization: Bearer <user_token>" http://localhost:8080/system/state

# ✅ Can view profile
curl -H "Authorization: Bearer <user_token>" http://localhost:8080/auth/profile

# ❌ Cannot start pomodoro (403 Forbidden)
curl -X POST http://localhost:8080/system/start \
  -H "Authorization: Bearer <user_token>"
```

**ADMIN Role Permissions:**

```bash
# ✅ Can view system state
curl -H "Authorization: Bearer <admin_token>" http://localhost:8080/system/state

# ✅ Can view profile
curl -H "Authorization: Bearer <admin_token>" http://localhost:8080/auth/profile

# ✅ Can start pomodoro
curl -X POST http://localhost:8080/system/start \
  -H "Authorization: Bearer <admin_token>"
```

### Security Notes

- **Admin Registration**: The `/auth/register-admin` endpoint should be **disabled in production**
- **JWT Secret**: Change the default JWT secret in production environments
- **Password Hashing**: Uses bcrypt with cost factor 12 for secure password hashing
- **Input Validation**: All endpoints validate and sanitize input data
- **Error Handling**: Appropriate HTTP status codes (401 Unauthorized, 403 Forbidden)
- **Role Changes**: User roles are fetched from database on each request, ensuring immediate effect of role changes without requiring logout/login

### Role Change Behavior

**Before (Old Approach):**

- User role stored in JWT token
- Role changes require user to logout and login again
- Old tokens remain valid with outdated role information

**After (Current Approach):**

- User role fetched from database on each request
- Role changes take effect immediately on next API call
- No need for users to logout/login when roles change
- More secure and administratively convenient
