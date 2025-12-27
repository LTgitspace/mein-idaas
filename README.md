# Mein IDaaS - Personal Identity-as-a-Service

A lightweight, enterprise-free Identification-as-a-Service (IDaaS) platform built with Go and PostgreSQL. Strip away the bloatware and get a clean, minimal authentication and authorization system tailored for personal use.

## Overview

Mein IDaaS provides a self-hosted alternative to enterprise IDaaS solutions like Okta or Azure AD. Perfect for developers who want:

- **Simple JWT-based authentication** with access and refresh tokens
- **Token rotation** every 7 days with grace period for enhanced security
- **Email verification** with OTP codes (6-digit verification)
- **Role-based access control (RBAC)** with granular permissions
- **PostgreSQL persistence** with GORM ORM
- **RSA-256 JWT signing** for asymmetric key security
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
│  (Auth, Email, Verification)    │
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
├── README.md              # This file
│
├── controller/
│   ├── AuthController.go        # Register, Login, Refresh endpoints
│   └── VerificationController.go # Email verification endpoints
│
├── service/
│   ├── AuthService.go           # Core authentication & authorization
│   ├── VerificationService.go   # Email verification logic
│   └── EmailService.go          # Email sending (SMTP)
│
├── repository/
│   ├── UserRepository.go                    # User CRUD
│   ├── CredentialRepository.go              # Credential storage
│   ├── RefreshTokenRepository.go            # Refresh token management
│   ├── RoleRepository.go                    # Role queries
│   ├── VerificationRepository.go            # Verification code storage
│   └── InMemoryVerificationRepository.go    # In-memory OTP storage
│
├── model/
│   ├── User.go         # User entity with roles & email verification
│   ├── Credential.go   # Password credentials
│   ├── RefreshTokens.go # Refresh token storage with rotation
│   ├── Role.go         # Role definitions
│   ├── AuthClaims.go   # JWT claims structure
│   └── Enums.go        # Enum types
│
├── dto/
│   ├── Auth.go    # Auth request/response DTOs
│   └── Verify.go  # Verification request/response DTOs
│
├── util/
│   ├── env.go              # Environment variable loading
│   ├── DButil.go           # Database initialization
│   ├── Auth_helpers.go     # Password hashing & verification
│   ├── JWT_generator.go    # Token generation (RSA-256)
│   ├── token_verify.go     # Token validation
│   ├── RSA.go              # RSA key management
│   ├── OTP.go              # OTP generation
│   ├── CronJob.go          # Cleanup jobs
│   ├── Validator.go        # Input validation
│   └── Error.go            # Error utilities
│
├── seeder/
│   └── RoleSeeder.go  # Initialize default roles
│
├── docs/
│   ├── docs.go        # Swagger documentation
│   ├── swagger.json
│   └── swagger.yaml
│
├── config/
│   └── Configuration files
│
├── Dockerfile         # Docker image definition
├── docker-compose.yml # Docker Compose setup
└── LICENSE
```

## Authentication Flow

### Complete User Journey

```
1. REGISTER
   POST /api/v1/auth/register
   ├─ Create user account
   ├─ Hash password with bcrypt
   ├─ Assign default "user" role
   └─ Send verification email (async)

2. VERIFY EMAIL
   POST /api/v1/auth/verify
   ├─ Submit email + 6-digit OTP code
   ├─ Validate code (5-minute TTL)
   └─ Mark user as email-verified

3. LOGIN
   POST /api/v1/auth/login
   ├─ Check credentials
   ├─ Check if email is verified
   ├─ If not verified: send email, return 403
   ├─ If verified: issue tokens, return 200
   └─ Store refresh token hash in DB

4. USE ACCESS TOKEN
   GET /api/v1/protected
   ├─ Authorization: Bearer <access_token>
   ├─ Server validates JWT signature
   ├─ Extract user_id & roles from token
   └─ Process request

5. REFRESH TOKENS (every 7 days)
   POST /api/v1/auth/refresh
   ├─ Send refresh token cookie
   ├─ Validate token exists & not revoked
   ├─ Check for theft (grace period logic)
   ├─ Issue new token pair
   ├─ Revoke old refresh token
   └─ Return new access token

6. RESEND VERIFICATION
   POST /api/v1/auth/resend
   ├─ Email already registered
   └─ Send new OTP code
```

### Token Lifecycle

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
├─ Server validates & marks old as replaced
├─ Issues NEW Access Token (15 min)
├─ Issues NEW Refresh Token (7 days)
│
...Repeat every 7 days
```

### Token Rotation with Grace Period

The system implements a **10-second grace period** to handle network race conditions:

```
Normal Refresh (First time):
├─ Send refresh token (Token A)
├─ Server issues new pair
├─ Mark Token A as "replaced_at" = NOW()
├─ Return new Token B

Concurrent Request (within 10 seconds):
├─ Client retries with Token A again
├─ Server sees "replaced_at" is set
├─ Check: duration = NOW() - replaced_at < 10 seconds
├─ Result: Within grace period, return Token B (safe retry)

Replay Attack (after 10 seconds):
├─ Attacker uses old Token A
├─ Server sees duration > 10 seconds
├─ Result: REJECTED - "refresh token reuse detected"
└─ Account is locked for security
```

## API Endpoints

### Authentication Endpoints

#### 1. Register User
**POST** `/api/v1/auth/register`

**Request:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "SecurePassword123!"
}
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "John Doe",
  "email": "john@example.com"
}
```

**What Happens:**
- User account is created in database
- Password is hashed with bcrypt
- Default "user" role is assigned
- Verification email with OTP is sent (asynchronously)
- User is NOT yet logged in (must verify email first)

---

#### 2. Resend Verification Email
**POST** `/api/v1/auth/resend`

**Request:**
```json
{
  "email": "john@example.com"
}
```

**Response (202 Accepted):**
```json
{
  "message": "verification code sent"
}
```

**Status Codes:**
- 202 - Email sent successfully (async)
- 404 - User not found
- 400 - Invalid email format
- 500 - Failed to send email

---

#### 3. Verify Email with OTP
**POST** `/api/v1/auth/verify`

**Request:**
```json
{
  "email": "john@example.com",
  "code": "123456"
}
```

**Response (200 OK):**
```json
{
  "message": "email verified"
}
```

**Status Codes:**
- 200 - Email verified, account activated
- 401 - Invalid or expired OTP code
- 404 - User not found
- 400 - Invalid request format

**What Happens:**
- OTP code is validated (6 digits, 5-minute TTL)
- User's `isEmailVerified` flag is set to true
- User can now login

---

#### 4. Login
**POST** `/api/v1/auth/login`

**Request:**
```json
{
  "email": "john@example.com",
  "password": "SecurePassword123!"
}
```

**Response (200 OK) - Email Verified:**
```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "expires_in": 900
}
```

**Response (403 Forbidden) - Email Not Verified:**
```json
{
  "error": "email not verified",
  "message": "verification email has been sent to your email address"
}
```

**Headers Set:**
```
Set-Cookie: refresh_token=a1b2c3d4...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth
```

**What Happens:**
- Credentials are validated
- Email verification status is checked
- If NOT verified: verification email is sent, 403 returned
- If verified: tokens are issued, refresh token stored in HTTP-only cookie

---

#### 5. Refresh Tokens
**POST** `/api/v1/auth/refresh`

**Headers:**
```
Cookie: refresh_token=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
```

**Response (200 OK):**
```json
{
  "access_token": "new_access_token_here",
  "refresh_token": "new_refresh_token_here",
  "expires_in": 900
}
```

**Response (401 Unauthorized):**
```json
{
  "error": "refresh token reuse detected: account locked for security"
}
```

**Server Actions:**
- Validates refresh token exists & not revoked
- Checks 10-second grace period for concurrent requests
- Detects theft/replay attacks
- Generates new token pair
- Marks old token as "replaced"
- Issues new refresh token (7-day TTL)

---

#### 6. Health Check
**GET** `/health`

**Response (200 OK):**
```json
{
  "status": "ok"
}
```

---

## JWT Token Structure

### Access Token Payload
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "roles": ["user"],
  "iss": "mein-idaas",
  "aud": ["my-game-server", "smoking-app"],
  "iat": 1703247200,
  "exp": 1703248100
}
```

**Fields:**
- `sub` - Subject (user ID)
- `roles` - User's assigned roles
- `iss` - Issuer (mein-idaas)
- `aud` - Audience (can be multiple services)
- `iat` - Issued at (timestamp)
- `exp` - Expires at (15 minutes from issue)

### Refresh Token Storage (Database)
```
id: UUID
user_id: UUID
token_hash: bcrypt_hashed_token
expires_at: 2025-12-30T12:00:00Z
replaced_at: 2025-12-25T14:30:00Z (grace period marker)
replaced_by_token_id: UUID (points to new token)
revoked_at: null (null = active)
client_ip: 192.168.1.1
user_agent: Mozilla/5.0...
created_at: 2025-12-23T12:00:00Z
```

---

## Getting Started

### Prerequisites

- Go 1.25 or higher
- PostgreSQL 16 or higher
- Git
- (Optional) Docker & Docker Compose

### Installation

#### Option 1: Local Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/mein-idaas.git
cd mein-idaas
```

2. Install dependencies:
```bash
go mod download
```

3. Create `.env` file in project root:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_secure_password
DB_NAME=idaas_db

# JWT Configuration (RSA-256)
JWT_SECRET_KEY_PATH=./private_key.pem
JWT_PUBLIC_KEY_PATH=./public_key.pem

# Token TTL
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# Grace Period for Refresh Token Rotation
REFRESH_GRACE_PERIOD=10s

# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@mein-idaas.com

# Server Configuration
PORT=4000
COOKIE_PATH=/api/v1/auth
```

4. Generate RSA keys (if not present):
```bash
openssl genrsa -out private_key.pem 2048
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

5. Create PostgreSQL database:
```bash
psql -U postgres
CREATE DATABASE idaas_db;
\q
```

6. Run the application:
```bash
go run main.go
```

Server will start on `http://localhost:4000`

---

#### Option 2: Docker Setup

1. Clone the repository:
```bash
git clone https://github.com/yourusername/mein-idaas.git
cd mein-idaas
```

2. Create `.env` file (as above)

3. Start with Docker Compose:
```bash
docker-compose up -d
```

This will:
- Start PostgreSQL container
- Build and start Go API container
- Auto-migrate database schema
- Seed default roles

4. Access the API:
```
http://localhost:4000
```

---

### Verify Installation

Check server health:
```bash
curl http://localhost:4000/health
```

Expected response:
```json
{"status":"ok"}
```

---

## API Documentation

### Swagger UI

Access interactive API documentation:
```
http://localhost:4000/swagger/index.html
```

Full OpenAPI spec:
```
http://localhost:4000/swagger/doc.json
```

---

## Database Schema

### Users Table
```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  is_email_verified BOOLEAN DEFAULT false,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Fields:
- `id` - Unique identifier (UUID v4)
- `name` - User's display name
- `email` - Unique email address
- `is_email_verified` - Email verification status (false until OTP confirmed)
- `created_at` - Account creation timestamp
- `updated_at` - Last update timestamp

---

### Credentials Table
```sql
CREATE TABLE credentials (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type VARCHAR(50) NOT NULL,
  value TEXT NOT NULL,
  active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id, type)
);
```

Fields:
- `id` - Unique identifier
- `user_id` - Reference to user
- `type` - Credential type (currently 'password')
- `value` - Hashed credential value (Argon2id for passwords)
- `active` - Whether credential is active
- Unique constraint ensures one credential per type per user

---

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  replaced_at TIMESTAMP,
  replaced_by_token_id UUID REFERENCES refresh_tokens(id),
  revoked_at TIMESTAMP,
  client_ip VARCHAR(45),
  user_agent TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Fields:
- `id` - Token ID (UUID)
- `user_id` - Token owner
- `token_hash` - Bcrypt hash of the actual token (stored securely)
- `expires_at` - Token expiration (7 days from creation)
- `replaced_at` - When this token was rotated (marks grace period start)
- `replaced_by_token_id` - UUID of replacement token (for grace period retry)
- `revoked_at` - Manual revocation timestamp (null = active)
- `client_ip` - Client IP for audit trail
- `user_agent` - Browser/client identifier

**Grace Period Logic:**
- First use: `replaced_at` = null → Normal rotation
- Concurrent retry (< 10s): Use old `replaced_by_token_id` → Safe
- Replay attack (> 10s): Reject → Security locked

---

### Roles Table
```sql
CREATE TABLE roles (
  id UUID PRIMARY KEY,
  code VARCHAR(50) UNIQUE NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Default roles seeded:
- `admin` - Full system access
- `user` - Standard user role (default for new registrations)

---

### User Roles Junction Table
```sql
CREATE TABLE user_roles (
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, role_id)
);
```

Many-to-many relationship between users and roles.

---

### Verification Codes (In-Memory)
```
Structure: Map[userID]VerificationCode
{
  code: "123456" (6 digits)
  created_at: timestamp
  expires_at: timestamp (5 minutes from creation)
}
```

Note: Uses in-memory storage with automatic cleanup. For production, consider Redis or database persistence.

---

## Security Features

### Password Security
- Bcrypt hashing with configurable salt rounds (default: 10)
- Passwords never stored in plain text
- Constant-time comparison prevents timing attacks

### JWT Security
- RSA-256 asymmetric signing (public/private key pair)
- `sub` (subject) contains user ID
- `roles` included for quick authorization checks
- Separate access and refresh token lifecycles
- Short-lived access tokens (15 minutes)
- Long-lived refresh tokens (7 days)

### Email Verification
- 6-digit OTP codes (randomly generated)
- 5-minute expiration per code
- Automatic code cleanup
- Email sent asynchronously (non-blocking)

### Token Rotation & Grace Period
- Old refresh tokens marked as "replaced" on rotation
- 10-second grace period for network retry safety
- Theft detection after grace period expires
- Account lockdown on suspicious reuse
- Prevents token replay attacks

### Database Security
- Refresh tokens stored hashed (bcrypt), not plain text
- Unique email constraint prevents duplicate accounts
- Foreign key constraints with cascade deletion
- Token audit trail (IP, User-Agent, timestamps)
- Automatic cleanup of expired tokens (daily cron)

### HTTP Security
- Refresh tokens stored in HTTP-only cookies
- Secure flag set (HTTPS only in production)
- SameSite=Strict prevents CSRF attacks
- Access token in response body for client-side use

---

## Configuration

### Environment Variables

```env
# Database
DB_HOST              # PostgreSQL host (default: localhost)
DB_PORT              # PostgreSQL port (default: 5432)
DB_USER              # PostgreSQL user
DB_PASSWORD          # PostgreSQL password
DB_NAME              # Database name

# JWT / RSA Keys
JWT_SECRET_KEY_PATH  # Path to private_key.pem
JWT_PUBLIC_KEY_PATH  # Path to public_key.pem

# Token Lifetimes
JWT_ACCESS_TTL       # Access token TTL (default: 15m)
JWT_REFRESH_TTL      # Refresh token TTL (default: 168h = 7 days)

# Grace Period
REFRESH_GRACE_PERIOD # Grace window for token rotation (default: 10s)

# Email / SMTP
SMTP_HOST            # SMTP server host
SMTP_PORT            # SMTP server port
SMTP_USER            # SMTP username
SMTP_PASSWORD        # SMTP password (app password for Gmail)
SMTP_FROM            # From email address

# Server
PORT                 # Server port (default: 4000)
COOKIE_PATH          # Cookie path (default: /api/v1/auth)
```

---

## HTTP Status Codes

| Code | Status | Scenario |
|------|--------|----------|
| 200 | OK | Login/Refresh successful, verification email sent |
| 201 | Created | User registered successfully |
| 202 | Accepted | Verification email accepted for sending (async) |
| 400 | Bad Request | Invalid request format or validation error |
| 401 | Unauthorized | Invalid credentials, expired token, or token replay detected |
| 403 | Forbidden | Email not verified - can't login yet |
| 404 | Not Found | User or resource not found |
| 500 | Server Error | Database error or internal server error |

---

## Troubleshooting

### "email not verified"
- **Cause:** User tried to login without verifying email first
- **Solution:** User needs to submit 6-digit OTP via `/auth/verify` endpoint
- **How to resend:** POST `/auth/resend` with email address

### "refresh token reuse detected: account locked for security"
- **Cause:** Token was reused after 10-second grace period ended
- **Solution:** This is a security feature. User must login again with credentials
- **What happened:** Possible token theft detected

### "invalid or expired verification code"
- **Cause:** OTP code is wrong or older than 5 minutes
- **Solution:** Request new code via `/auth/resend` endpoint

### "email already in use"
- **Cause:** Another account with same email exists
- **Solution:** Use different email or request password reset (future feature)

### "invalid credentials"
- **Cause:** Email doesn't exist or password is wrong
- **Solution:** Check email spelling or use `/auth/register` to create account

### Database connection errors
- **Cause:** PostgreSQL not running or connection string wrong
- **Solution:** Verify database is running and `.env` file has correct credentials
- **Check:** `psql -h localhost -U postgres -d idaas_db`

### Swagger not loading
- **Cause:** Swagger docs not generated
- **Solution:** Run `swag init` in project root
- **Or:** Restart server - docs auto-generate on startup

### RSA keys not found
- **Cause:** `private_key.pem` and `public_key.pem` missing
- **Solution:** Generate with:
```bash
openssl genrsa -out private_key.pem 2048
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `gofiber/fiber/v2` | Ultra-fast HTTP framework |
| `golang-jwt/jwt/v5` | JWT signing & verification |
| `gorm.io/gorm` | ORM for database operations |
| `gorm.io/driver/postgres` | PostgreSQL driver for GORM |
| `golang.org/x/crypto` | Bcrypt password hashing |
| `google/uuid` | UUID v4 generation |
| `gofiber/swagger` | Swagger UI integration |
| `swaggo/swag` | Swagger documentation generation |
| `joho/godotenv` | .env file loading |
| `gomail.v2` | Email sending (SMTP) |

---

## Development

### Running Tests

```bash
go test ./...
```

### Generating Swagger Docs

```bash
swag init
```

This generates/updates:
- `docs/docs.go`
- `docs/swagger.json`
- `docs/swagger.yaml`

### Code Structure Explained

**Controllers** - `controller/` folder
- HTTP request/response handling
- Status code and error responses
- Swagger annotations for documentation

**Services** - `service/` folder
- Business logic and workflows
- Orchestrates repositories and utilities
- Handles token generation, email sending, etc.

**Repositories** - `repository/` folder
- Interface-based for dependency injection
- CRUD operations abstracted
- Database query logic isolated

**DTOs** - `dto/` folder
- Data Transfer Objects
- Request/response serialization
- Decouples API contracts from models

**Models** - `model/` folder
- Database entities with GORM tags
- Relationships: User → Roles, Credentials, RefreshTokens
- Enums for credential types

**Utilities** - `util/` folder
- Reusable helper functions
- JWT generation and validation
- Password hashing and OTP generation
- Database initialization
- Environment variable loading

---

## Future Enhancements

- [ ] Multi-factor authentication (MFA) with TOTP
- [ ] OAuth2/OIDC provider mode
- [ ] Permission-based authorization (beyond roles)
- [ ] Password reset flow
- [ ] Account lockout after failed attempts
- [ ] Audit logging and compliance tracking
- [ ] Session management and device tracking
- [ ] Redis caching for OTP and sessions
- [ ] Rate limiting per user/IP
- [ ] HTTPS/TLS enforcement
- [ ] Token introspection endpoint
- [ ] User profile management
- [ ] Admin dashboard

---

## License

MIT License - See LICENSE file for details

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Submit a pull request

---

## Support

For issues, questions, or suggestions, please open an issue on GitHub.

---

Built with simplicity and control in mind for developers who want to own their authentication infrastructure.
