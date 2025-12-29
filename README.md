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

#### 7. Send Password Change OTP
**POST** `/api/v1/auth/password-change/send-otp`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response (200 OK):**
```json
{
  "message": "OTP sent to your email",
  "email": "john@example.com"
}
```

**Status Codes:**
- 200 - OTP sent successfully
- 401 - Invalid or missing access token
- 404 - User not found
- 500 - Failed to send email

**What Happens:**
- Validates access token and extracts user ID
- Generates 6-digit OTP code
- Sends OTP to user's registered email
- OTP valid for 5 minutes

---

#### 8. Change Password
**POST** `/api/v1/auth/password-change`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Request:**
```json
{
  "old_password": "OldPassword123!",
  "new_password": "NewPassword456!",
  "otp_code": "123456"
}
```

**Response (200 OK):**
```json
{
  "message": "password changed successfully",
  "email": "john@example.com"
}
```

**Status Codes:**
- 200 - Password changed successfully
- 400 - Invalid OTP or passwords don't meet requirements
- 401 - Invalid/expired token or wrong old password
- 500 - Internal server error

**What Happens:**
- Validates access token
- Verifies OTP code (must be valid and not expired)
- Validates old password is correct
- Ensures new password is different from old
- Hashes new password with Argon2
- Updates password credential in database
- OTP is consumed and cannot be reused

---

#### 9. Send Forgot Password OTP
**POST** `/api/v1/auth/forgot-password/send-otp`

**Request:**
```json
{
  "email": "john@example.com"
}
```

**Response (200 OK):**
```json
{
  "message": "if email exists, a password reset code has been sent"
}
```

**Status Codes:**
- 200 - Always returns 200 (even if email doesn't exist)
- 400 - Invalid email format
- 500 - Failed to send email

**Security Note:** Returns 200 regardless of whether email exists in system. This prevents email enumeration attacks.

**What Happens:**
- Checks if email exists in system
- If NOT found: Silently logs the request and returns success
- If found: Generates 6-digit OTP code with 5-minute TTL
- Sends OTP to user's email
- OTP stored securely with user ID as key

---

#### 10. Reset Password with OTP
**POST** `/api/v1/auth/forgot-password/reset`

**Request:**
```json
{
  "email": "john@example.com",
  "otp": "123456"
}
```

**Response (200 OK):**
```json
{
  "message": "password has been reset, check your email for the temporary password",
  "email": "john@example.com"
}
```

**Response (400 Bad Request):**
```json
{
  "error": "invalid or expired OTP code"
}
```

**Status Codes:**
- 200 - Password reset successfully
- 400 - Invalid/expired OTP code
- 404 - User not found
- 500 - Internal server error

**What Happens:**
- Validates email exists in system
- Verifies OTP code (6 digits, 5-minute expiration)
- Generates random 8-character temporary password
- Hashes temporary password with Argon2
- Updates user's password credential
- Sends temporary password to user's email
- OTP is consumed and deleted (prevents reuse)
- User can now login with temporary password

**Important:** User should change the temporary password immediately after login for security.

---

## Complete Authentication Flows

### Registration & Email Verification
```
1. User calls POST /auth/register
   ├─ System creates user account
   ├─ System hashes password (Argon2)
   ├─ System sends verification email with OTP (async)
   └─ User receives OTP in email

2. User calls POST /auth/verify
   ├─ System validates OTP (6 digits, 5-minute TTL)
   ├─ System marks user as email-verified
   └─ User can now login

3. User calls POST /auth/login
   ├─ System validates credentials
   ├─ System checks email is verified
   ├─ System issues access token (15 minutes)
   ├─ System issues refresh token (7 days, stored in HTTP-only cookie)
   └─ User receives tokens and is logged in
```

### Password Change (Authenticated User)
```
1. User calls POST /auth/password-change/send-otp
   ├─ System validates access token
   ├─ System generates 6-digit OTP
   ├─ System sends OTP to user's email
   └─ User receives OTP

2. User calls POST /auth/password-change
   ├─ System validates access token
   ├─ System validates OTP (6 digits, 5-minute TTL)
   ├─ System verifies old password is correct
   ├─ System hashes new password (Argon2)
   ├─ System updates password in database
   ├─ System deletes used OTP (prevents replay)
   └─ Password is changed successfully
```

### Password Reset (Forgot Password)
```
1. User calls POST /auth/forgot-password/send-otp
   ├─ User provides email address
   ├─ System checks if email exists (silently logs if not)
   ├─ System generates 6-digit OTP (5-minute TTL)
   ├─ System sends OTP to email
   └─ User receives OTP (if account exists)

2. User calls POST /auth/forgot-password/reset
   ├─ User provides email + OTP code
   ├─ System validates OTP is correct and not expired
   ├─ System generates random 8-character temporary password
   ├─ System hashes temporary password (Argon2)
   ├─ System updates password in database
   ├─ System sends temporary password to email
   ├─ System deletes used OTP
   └─ User can now login with temporary password

3. User calls POST /auth/login
   ├─ User logs in with email + temporary password
   ├─ System issues tokens
   └─ User is logged in

4. User calls POST /auth/password-change/send-otp
   ├─ User changes temporary password to permanent one (recommended)
   └─ Password is secured
```

### Token Rotation (Every 7 Days)
```
Day 0: User logs in
├─ Access Token issued (15 minutes)
├─ Refresh Token issued (7 days, stored in HTTP-only cookie)
└─ Old refresh token stored hashed in DB

Day 7: Access token expires during API call
├─ Client calls POST /auth/refresh
├─ Client sends refresh token cookie
├─ Server validates refresh token exists & not revoked
├─ Server checks 10-second grace period for concurrent requests
├─ Server detects no theft (first time using token)
├─ Server issues NEW access token (15 minutes)
├─ Server issues NEW refresh token (7 days)
├─ Server marks old token as "replaced"
└─ Process repeats every 7 days...
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

# Argon2 Password Hashing
ARGON2_TIME=3
ARGON2_MEMORY=65536
ARGON2_THREADS=4
ARGON2_KEY_LENGTH=32
ARGON2_SALT_LENGTH=16
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
- **Argon2id hashing** - Memory-hard password hashing algorithm resistant to GPU/ASIC attacks
- **Dynamic parameter support** - Argon2 parameters configurable via environment variables without locking users out
- **Parameter extraction** - Each password hash stores its own parameters (time, memory, threads)
- **Safe parameter changes** - Global parameters can be updated; old passwords validated using their stored parameters
- **Passwords never stored in plain text** - Only secure hashes stored in database
- **Constant-time comparison** - Prevents timing attacks during password verification

### Argon2 Implementation Details

Your system uses **Argon2id** with the following hash format:

```
$argon2id$v=19$m=<memory>,t=<time>,p=<threads>$<salt_hex>$<hash_hex>
```

**Example hash:**
```
$argon2id$v=19$m=65536,t=3,p=4$abcd1234efgh5678$xyz123abc456def789xyz123abc456def789xyz1234567
```

**How it works:**

1. **Hash Creation** - New passwords hashed with current global parameters from `.env`:
   - Parameters embedded in the hash string
   - Salt randomly generated and stored in hash
   - Hash is computed using embedded parameters

2. **Password Validation** - When user logs in:
   - Extract salt from stored hash
   - Extract parameters (m, t, p) from hash format
   - Recompute hash using **EXTRACTED parameters** (not global variables)
   - Compare computed hash with stored hash
   - User logs in successfully

3. **Safe Parameter Updates** - You can change Argon2 parameters in `.env`:
   - Old passwords continue to validate (use their stored parameters)
   - New passwords use updated parameters
   - No users locked out during migration
   - Gradual security improvement as users re-register

**Why this is secure:**
- Each password is self-contained with its own parameters
- Global parameters only affect NEW passwords
- Old password hashes are immutable and always validate correctly
- Allows incremental security upgrades without service disruption

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
- Refresh tokens stored hashed (SHA-256), not plain text
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

# Argon2 Password Hashing
ARGON2_TIME          # Iterations (default: 3) - Higher = more secure but slower
ARGON2_MEMORY        # Memory in KB (default: 65536 = 64MB) - Higher = more resistant to attacks
ARGON2_THREADS       # Parallel threads (default: 4) - Should match CPU cores
ARGON2_KEY_LENGTH    # Hash output length in bytes (default: 32) - Higher = more secure
ARGON2_SALT_LENGTH   # Salt length in bytes (default: 16) - Higher = more unique

# Email / SMTP
SMTP_HOST            # SMTP server host
SMTP_PORT            # SMTP server port
SMTP_USER            # SMTP username
SMTP_PASSWORD        # SMTP password (app password for Gmail)
SMTP_SENDER_NAME     # Sender name in emails

# Server
PORT                 # Server port (default: 4000)
COOKIE_PATH          # Cookie path (default: /api/v1/auth)
```

### Argon2 Parameter Tuning

Choose parameters based on your security requirements and hardware:

**Development/Testing (Fast):**
```env
ARGON2_TIME=1
ARGON2_MEMORY=16384
ARGON2_THREADS=2
```
- Password hashing: ~50ms
- Login: ~100-150ms

**Standard Security (Recommended):**
```env
ARGON2_TIME=3
ARGON2_MEMORY=65536
ARGON2_THREADS=4
```
- Password hashing: ~200-300ms
- Login: ~400-500ms

**High Security (Production):**
```env
ARGON2_TIME=4
ARGON2_MEMORY=262144
ARGON2_THREADS=8
```
- Password hashing: ~1-2 seconds
- Login: ~2-3 seconds

Important: Higher parameters = slower login (users notice this). Choose based on your threat model.

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
