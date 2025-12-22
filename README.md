# Mein IDaaS - Personal Identity-as-a-Service

A lightweight, enterprise-free Identification-as-a-Service (IDaaS) platform built with Go and PostgreSQL. Strip away the bloatware and get a clean, minimal authentication and authorization system tailored for personal use.

## Overview

Mein IDaaS provides a self-hosted alternative to enterprise IDaaS solutions like Okta or Azure AD. Perfect for developers who want:

- **Simple JWT-based authentication** with access and refresh tokens
- **Token rotation** every 7 days for enhanced security
- **Role-based access control (RBAC)** with granular permissions
- **PostgreSQL persistence** with GORM ORM
- **Built-in Swagger documentation** for easy API exploration
- **Zero enterprise complexity** - just what you need

## Architecture

The project follows a clean layered architecture:

```
┌─────────────────────────────────┐
│      HTTP Controllers           │ (Fiber Framework)
│   (Request/Response Handling)   │
└──────────────┬──────────────────┘
               │
┌──────────────▼──────────────────┐
│     Service Layer               │ (Business Logic)
│  (Auth, Token Management)       │
└──────────────┬──────────────────┘
               │
┌──────────────▼──────────────────┐
│     Repository Layer            │ (Data Access)
│  (CRUD Operations, Queries)     │
└──────────────┬──────────────────┘
               │
┌──────────────▼──────────────────┐
│     PostgreSQL Database         │
└─────────────────────────────────┘
```

## Project Structure

```
mein-idaas/
├── main.go                 # Application entry point & route setup
├── go.mod & go.sum        # Dependency management
├── refresh_token_flow.md  # Token rotation flow documentation
│
├── controller/            # HTTP handlers (Fiber)
│   └── AuthController.go # Register, Login, Refresh endpoints
│
├── service/              # Business logic
│   └── AuthService.go    # Core authentication & authorization
│
├── repository/           # Data access layer
│   └── Repository.go     # User, Credential, RefreshToken repositories
│
├── model/               # Database models
│   ├── User.go         # User entity with roles
│   ├── Credential.go   # Password credentials
│   ├── RefreshToken.go # Refresh token storage
│   ├── Role.go         # Role definitions
│   └── AuthClaims.go   # JWT claims structure
│
├── dto/                # Data Transfer Objects
│   ├── Auth.go         # Request/Response DTOs
│   └── AuthClaims.go   # Claims DTO
│
├── util/               # Utility functions
│   ├── env.go         # Environment variable loading
│   ├── DButil.go      # Database initialization
│   ├── auth_helpers.go # Password hashing & verification
│   ├── JWT_generator.go # Token generation
│   └── token_verify.go  # Token validation
│
├── seeder/            # Database seeding
│   └── RoleSeeder.go  # Initialize default roles
│
├── docs/              # Swagger documentation (auto-generated)
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
│
├── config/            # Configuration files
└── LICENSE
```

## Authentication Flow

### Token Lifecycle

The system implements **JWT rotation with 7-day refresh cycles**:

```
Day 0 - User Logs In
├─ Access Token (15 minutes) ────────┐
│                                     │
├─ Refresh Token (7 days) ───────────┤
│   └─ Stored hashed in DB           │
│   └─ Can be revoked                │
│                                     │
Day 0-7: Use Access Token for API calls
│
Day 7: Access Token Expires
├─ Client sends Refresh Token
├─ Server validates & revokes old token
├─ Issues NEW Access Token (15 min)
├─ Issues NEW Refresh Token (7 days)
│
...Repeat every 7 days
```

### 1. Register (`POST /api/v1/auth/register`)

**Request:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "secure_password_123"
}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "John Doe",
  "email": "john@example.com"
}
```

**What happens:**
- User is created with default "user" role
- Password is hashed using bcrypt
- User can now log in

---

### 2. Login (`POST /api/v1/auth/login`)

**Request:**
```json
{
  "email": "john@example.com",
  "password": "secure_password_123"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "expires_in": 900
}
```

**Access Token Contains:**
- `user_id` - UUID of the user
- `email` - User's email
- `roles` - Assigned roles
- `exp` - Expiration time (15 minutes from now)

**Refresh Token:**
- Random 32-character string
- Stored hashed in database
- Valid for 7 days
- Tracked with client IP & user agent

---

### 3. Using Access Token

**Protected API Call:**
```bash
curl -H "Authorization: Bearer <access_token>" \
     http://localhost:4000/api/v1/profile
```

Server validates JWT without database lookup (stateless).

---

### 4. Refresh Tokens (`POST /api/v1/auth/refresh`)

**Request:**
```json
{
  "refresh_token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
}
```

**Response:**
```json
{
  "access_token": "new_token_here",
  "refresh_token": "new_refresh_token_here",
  "expires_in": 900
}
```

**Server Actions:**
-  Validates refresh token exists in database
-  Checks expiration & revocation status
-  Generates new token pair
-  **Revokes old refresh token** (rotation)
-  Tracks client context (IP, User-Agent)

---

##  Getting Started

### Prerequisites

- **Go 1.25+**
- **PostgreSQL 16+**
- **Git**

### Installation

1. **Clone the repository:**
```bash
git clone https://github.com/yourusername/mein-idaas.git
cd mein-idaas
```

2. **Install dependencies:**
```bash
go mod download
```

3. **Set up environment variables:**

Create a `.env` file in the project root:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=idaas_db

# JWT
JWT_SECRET=your_super_secret_key_min_32_chars_long
JWT_EXPIRY_MINUTES=15
REFRESH_TOKEN_EXPIRY_DAYS=7

# Server
SERVER_PORT=4000
```

4. **Initialize the database:**

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE idaas_db;

# Exit psql
\q
```

5. **Run the application:**

```bash
go run main.go
```

The server will:
- Connect to PostgreSQL
- Auto-migrate database schema
- Seed default roles (admin, user)
- Start on `http://localhost:4000`

---

##  API Documentation

### Swagger UI

Access interactive API docs at:
```
http://localhost:4000/swagger/index.html
```

### Endpoints

#### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/auth/register` | Register new user |
| `POST` | `/api/v1/auth/login` | Login with credentials |
| `POST` | `/api/v1/auth/refresh` | Refresh token pair |

#### Health Check

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Server health check |

---

##  Security Features

### Password Security
- **Bcrypt hashing** with salt rounds
- Passwords never stored in plain text
- Constant-time comparison prevents timing attacks

### JWT Security
- **HS256 algorithm** for token signing
- `user_id` and `role` embedded in token for quick identification
- Separate access and refresh token lifecycles
- Short-lived access tokens (15 minutes)
- Long-lived refresh tokens (7 days)

### Token Rotation
- Old refresh tokens are **automatically revoked** on rotation
- Prevents token replay attacks
- Client context (IP, User-Agent) is tracked

### Database Security
- Refresh tokens stored **hashed**, not plain text
- Unique user email constraint
- Foreign key constraints with cascade deletion
- Soft-delete ready (via `UpdatedAt`)

---

## 🛠️ Development

### Running Tests

```bash
go test ./...
```

### Code Structure

**DTOs (Data Transfer Objects)** - `dto/` folder
- Handles JSON serialization/deserialization
- Decouples API contracts from models

**Models** - `model/` folder
- Database entities with GORM tags
- Relationships: User → Roles, Credentials, RefreshTokens

**Repositories** - `repository/` folder
- Interface-based for dependency injection
- CRUD operations abstracted from business logic

**Services** - `service/` folder
- Core business logic (Register, Login, Refresh)
- Token generation & validation
- Role assignment

**Controllers** - `controller/` folder
- HTTP request/response handling
- Route handlers with proper status codes
- Swagger annotations for documentation

---

## Token Claim Structure

### Access Token Payload
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "john@example.com",
  "roles": ["user"],
  "iat": 1703247200,
  "exp": 1703248100
}
```

### Refresh Token Storage (Database)
```json
{
  "id": "uuid",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "token_hash": "bcrypt_hashed_token",
  "expires_at": "2024-12-29T12:00:00Z",
  "revoked_at": null,
  "client_ip": "192.168.1.1",
  "user_agent": "Mozilla/5.0..."
}
```

---

## Refresh Token Rotation (7-Day Cycle)

### Why Every 7 Days?

The 7-day refresh cycle balances:
- **Security** - Limits exposure window for stolen tokens
- **User Experience** - Users don't need to re-login frequently
- **Audit Trail** - Every 7 days, you see new token generation

### How It Works

```
Day 0: User logs in
├─ Issue: Access Token (15 min) + Refresh Token (7 days)
│
Day 3: User makes API call with old Access Token
├─ Access Token is valid ✓
├─ API call succeeds
│
Day 4: Access Token expires, user calls /refresh
├─ Validate old Refresh Token exists & not revoked
├─ Issue: NEW Access Token (15 min) + NEW Refresh Token (7 days)
├─ Revoke: OLD Refresh Token (set revoked_at = NOW())
│
Day 7 (from original login): Can't use original Refresh Token
├─ User must login again with credentials
├─ NEW token cycle begins
```

---

## ️ Database Schema

### Users Table
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  is_email_verified BOOLEAN DEFAULT false,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### Credentials Table
```sql
CREATE TABLE credentials (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type VARCHAR(50) NOT NULL,  -- 'password', 'totp', etc.
  value TEXT NOT NULL,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(255) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  revoked_at TIMESTAMP,  -- NULL if not revoked
  client_ip VARCHAR(45),
  user_agent TEXT,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### Roles Table
```sql
CREATE TABLE roles (
  id UUID PRIMARY KEY,
  code VARCHAR(50) UNIQUE NOT NULL,  -- 'admin', 'user'
  description TEXT,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### User Roles Junction Table
```sql
CREATE TABLE user_roles (
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, role_id)
);
```

---

## Troubleshooting

### "invalid or unknown refresh token"
- **Cause:** Token doesn't exist in database or was never stored
- **Solution:** Ensure refresh tokens are being persisted in the database after login

### "refresh token expired or revoked"
- **Cause:** Token has been revoked or exceeded 7-day expiry
- **Solution:** User needs to login again with credentials

### Swagger not loading
- **Cause:** Swagger docs not generated
- **Solution:** Ensure `docs/` folder exists and `swag init` has been run

### Database connection errors
- **Cause:** PostgreSQL not running or credentials incorrect
- **Solution:** Verify DB connection string in environment variables

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `gofiber/fiber` | Ultra-fast HTTP framework |
| `golang-jwt/jwt` | JWT signing & verification |
| `gorm.io/gorm` | ORM for database operations |
| `gorm.io/driver/postgres` | PostgreSQL driver |
| `golang.org/x/crypto` | Password hashing (bcrypt) |
| `google/uuid` | UUID generation |
| `gofiber/swagger` | Swagger UI integration |
| `swaggo/swag` | Swagger documentation generation |

---

## Status Codes

| Code | Meaning |
|------|---------|
| `201` | User created successfully |
| `200` | Login/Refresh successful |
| `400` | Invalid request payload |
| `401` | Invalid credentials or expired token |
| `500` | Internal server error |

---

## Future Enhancements

- [ ] Email verification
- [ ] Multi-factor authentication (MFA)
- [ ] OAuth2/OIDC provider mode
- [ ] Permission-based authorization
- [ ] Audit logging
- [ ] Session management
- [ ] Rate limiting
- [ ] HTTPS/TLS enforcement

---

## License


---

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Submit a pull request

---

##  Support

For issues, questions, or suggestions, please open an issue on GitHub.

---

**Built with hate for developers who value simplicity and control.**
