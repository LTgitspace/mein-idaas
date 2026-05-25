# Mein IDaaS — Self-Hosted Identity-as-a-Service

A lightweight, self-hosted authentication and identity management server built with Go, Fiber, and PostgreSQL. Designed as a capstone project demonstrating modern auth infrastructure: JWT with RSA-256, refresh token rotation with theft detection, TOTP-based MFA, email verification, and role-based access control.

## Features

| Feature | Implementation |
|---|---|
| **User Registration & Login** | Argon2id password hashing, email/password auth |
| **Email Verification** | 6-digit OTP via SMTP, 5-minute TTL, in-memory store |
| **JWT Authentication** | RS256 (asymmetric RSA key pair), 15-min access + 7-day refresh tokens |
| **Refresh Token Rotation** | Automatic rotation with reuse detection, grace period, and theft lockdown |
| **MFA (TOTP)** | QR code enrollment, compatible with Google/Microsoft Authenticator, Authy |
| **Password Management** | Change password (old password + OTP), forgot password reset flow |
| **RBAC** | Role-based access control with user-role many-to-many mapping |
| **Rate Limiting** | 10 req/s per IP, 10-minute auto-ban on violation |
| **Request Metrics** | Every request logged with method, path, status, and duration |
| **API Docs** | Auto-generated Swagger/OpenAPI at `/swagger/` |
| **Docker Support** | Multi-stage Dockerfile + docker-compose with PostgreSQL 16 |

## Architecture

```
┌─────────────────────────────────┐
│         HTTP Layer              │
│   Fiber (controllers, routes)    │
├─────────────────────────────────┤
│       Middleware Layer          │
│   RateLimiter, TimerMetrics      │
├─────────────────────────────────┤
│        Service Layer            │
│   Auth, Verification, Email     │
├─────────────────────────────────┤
│      Repository Layer           │
│   User, Credential, Token, Role │
├─────────────────────────────────┤
│          Database               │
│   PostgreSQL (via GORM)         │
└─────────────────────────────────┘
```

Technology choices:
- **Go 1.25** — statically typed, compiled language
- **Fiber v2** — high-performance HTTP framework (fasthttp-based)
- **GORM** — ORM for PostgreSQL with auto-migration
- **RS256 JWT** — asymmetric signing (no shared secret)
- **Argon2id** — OWASP-recommended password hashing
- **Docker** — reproducible builds and deployments

## Project Structure

```
mein-idaas/
├── main.go                     # Entry point, route registration, DI wiring
├── go.mod / go.sum             # Dependency management
├── Dockerfile                  # Multi-stage build (Alpine)
├── docker-compose.yml          # App + PostgreSQL orchestration
│
├── controller/
│   ├── AuthController.go       # Register, login, refresh, MFA, password flows
│   └── VerificationController.go # Email verification endpoints
│
├── service/
│   ├── AuthService.go          # Core auth business logic (440+ lines)
│   ├── VerificationService.go  # OTP generation, storage, validation
│   ├── EmailService.go         # SMTP email sending (gomail v2)
│   └── LinkCredentialService.go # External OAuth credential linking (WIP)
│
├── repository/
│   ├── Repository.go           # Interface definitions
│   ├── UserRepository.go       # User CRUD
│   ├── CredentialRepository.go # Password/credential storage
│   ├── RefreshTokenRepository.go # Token lifecycle (create, update, delete expired)
│   ├── RoleRepository.go       # Role queries and seeding
│   └── InMemoryVerificationRepository.go # OTP cache with TTL
│
├── model/
│   ├── User.go                 # User entity (UUID, MFA fields, email verified flag)
│   ├── Credential.go           # Typed credentials (password, google, github, etc.)
│   ├── RefreshTokens.go        # Token with rotation family tree tracking
│   ├── Role.go                 # Role entity (code, name, system flag)
│   └── Enums.go                # CredentialType enum (password, google, facebook, github, zalo)
│
├── dto/
│   ├── Auth.go                 # Register, login, refresh, MFA, password DTOs
│   ├── AuthClaims.go           # JWT claims structure (roles + registered claims)
│   └── Verify.go               # Email verification DTOs
│
├── middleware/
│   ├── RateLimiter.go          # IP-based rate limiting with ban storage
│   └── TimerMetrics.go         # Request duration logging
│
├── util/
│   ├── Auth_helpers.go         # Argon2id hashing & password comparison
│   ├── DButil.go               # PostgreSQL connection initialization
│   ├── JWT_generator.go        # Token pair generation (RS256)
│   ├── token_verify.go         # JWT parsing and validation
│   ├── RSA.go                  # RSA key loading from environment variables
│   ├── OTP.go                  # Random digit code generation
│   ├── TOTP.go                 # TOTP secret + QR code generation (pquerna/otp)
│   ├── CronJob.go              # Daily expired token cleanup
│   ├── Validator.go            # Input validation (go-playground/validator)
│   ├── Error.go                # Error response helpers
│   └── env.go                  # Environment variable helpers
│
├── seeder/
│   └── RoleSeeder.go           # Seeds default roles (user, admin)
│
├── docs/                       # Auto-generated Swagger specs
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
│
└── config/                     # Reserved for future config files
```

## Data Model

### Entity Relationships

```
User (1) ────────< Credential (many)     — typed credentials per user
User (1) ────────< RefreshToken (many)   — token family tree
User (M) ────────> Role (N)             — many-to-many via user_roles
```

### Key Fields

**User** — `id` (UUID), `name`, `email` (unique), `is_email_verified`, `is_mfa_enabled`, `mfa_secret`, `backup_codes`, timestamps

**Credential** — `id`, `user_id`, `type` (enum: password/google/facebook/github/zalo), `value` (hashed), `active`

**RefreshToken** — `id`, `user_id`, `token_hash` (SHA-256), `expires_at`, `replaced_at` (rotation timestamp), `replaced_by_token_id` (family link), `revoked_at`, `client_ip`, `user_agent`

**Role** — `id`, `code` (unique), `name`, `description`, `is_system`

## Authentication Flow

### 1. Registration
```
POST /api/v1/auth/register
├── Validates input (name 2-50 chars, email format, password 8-72 chars)
├── Creates user with Argon2id-hashed password
├── Assigns default "user" role
├── Sends verification email with 6-digit OTP (async)
└── Returns 201 with user ID, name, email
```

### 2. Email Verification
```
POST /api/v1/auth/verify
├── Validates email + 6-digit code
├── Checks in-memory store (5-minute TTL)
├── Marks is_email_verified = true
└── Deletes used code (prevents replay)
```

### 3. Login
```
POST /api/v1/auth/login
├── Looks up user by email
├── Finds password credential (type: "password")
├── Verifies with Argon2id comparison
├── Checks is_email_verified (returns 403 + resends OTP if false)
├── Generates RS256 JWT access token (15 min) + refresh token (7 days)
├── Stores refresh token hash in DB (SHA-256)
└── Returns token pair + expires_in in JSON body
```

### 4. Token Refresh (with Rotation Detection)
```
POST /api/v1/auth/refresh
├── Parses refresh token JWT → extracts user_id + token_id
├── Loads token from DB by ID
├── Security checks: user match, not revoked, not expired

├── FIRST USE (replaced_at is null):
│   ├── Generates NEW access + refresh token pair
│   ├── Saves new token to DB
│   ├── Marks old token: replaced_at = now(), replaced_by_token_id = new_id
│   └── Returns new pair

├── GRACE PERIOD RETRY (within 10s of replacement):
│   ├── Detected: replaced_at is set, within grace window
│   ├── Finds child token via replaced_by_token_id
│   ├── Issues ONLY a new access token (reuses existing child refresh token)
│   └── Returns access token + re-signed child refresh token

└── THEFT DETECTED (replay after grace period):
    └── Returns 401: "refresh token reuse detected: account locked for security"
```

### 5. MFA Setup (TOTP)
```
POST /api/v1/auth/mfa/setup
├── Generates TOTP secret (base32)
├── Returns secret + QR code URL

GET /api/v1/auth/mfa/qrcode?email=...&secret=...
├── Returns QR code as PNG image (256x256)

GET /api/v1/auth/mfa/qrcode/base64?email=...&secret=...
├── Returns QR code as base64-encoded JSON

POST /api/v1/auth/mfa/confirm
├── Validates TOTP token against secret
├── Saves secret to user record
├── Sets is_mfa_enabled = true
└── Returns success
```

### 6. Password Change
```
POST /api/v1/auth/password-change/send-otp
├── Sends 6-digit OTP to user's email

POST /api/v1/auth/password-change
├── Validates old password + OTP + new password
├── Hashes new password (Argon2id)
├── Updates credential in DB
└── Returns success
```

### 7. Forgot Password Reset
```
POST /api/v1/auth/forgot-password/send-otp
├── Always returns 200 (prevents email enumeration)
├── Silently logs if email not found
└── Sends OTP if account exists

POST /api/v1/auth/forgot-password/reset
├── Validates email + OTP
├── Generates random 8-char temporary password
├── Hashes and updates credential
├── Emails temporary password to user
└── Returns success (user should change password after login)
```

## Configuration

### Environment Variables

All configuration is done via environment variables (no config files). Create a `.env` file:

```env
# ── Application ─────────────────────────────
PORT=4000
APP_NAME=mein-idaas
ENV=development                  # "production" enables strict TLS for SMTP

# ── Database ────────────────────────────────
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_db_password
DB_NAME=idaas
DB_SSLMODE=disable

# ── RSA Keys (PEM format) ──────────────────
# Keys are loaded as PEM strings from env vars, NOT file paths.
# Use \n for line breaks, or literal newlines if supported.
RSA_PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAK...\n-----END RSA PRIVATE KEY-----
RSA_PUBLIC_KEY=-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkq...\n-----END PUBLIC KEY-----

# ── JWT ────────────────────────────────────
JWT_ACCESS_TTL=15m               # Access token lifetime
JWT_REFRESH_TTL=168h             # Refresh token lifetime (7 days)
REFRESH_GRACE_PERIOD=10s         # Concurrency safety window
JWT_ISSUER=mein-idaas

# ── Argon2id Password Hashing ──────────────
ARGON2_TIME=3                    # Iterations
ARGON2_MEMORY=65536              # Memory in KB (64 MB)
ARGON2_THREADS=4                 # Parallelism
ARGON2_KEY_LENGTH=32             # Hash output bytes
ARGON2_SALT_LENGTH=16            # Salt bytes

# ── SMTP (Email) ───────────────────────────
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password      # Gmail: use App Password with 2FA
SMTP_SENDER_NAME=Mein IDaaS
```

### Generating RSA Keys

```bash
# Generate private key
openssl genrsa -out private_key.pem 2048

# Convert to PKCS#8 (if needed)
openssl pkcs8 -topk8 -nocrypt -in private_key.pem -out private_key_pkcs8.pem

# Extract public key
openssl rsa -in private_key.pem -pubout -out public_key.pem
```

The application supports both **PKCS#1** and **PKCS#8** private key formats. Key content (including `-----BEGIN/END-----` markers) must be provided as the full PEM string in the environment variable. Use `\n` to represent newlines if your environment doesn't support literal line breaks.

### SMTP Configuration Examples

**Gmail** (requires App Password with 2FA enabled):
```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=you@gmail.com
SMTP_PASS=your-16-char-app-password
```

**Self-hosted / other provider:**
```env
SMTP_HOST=mail.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASS=your-password
```

TLS certificate validation: In `development` mode (`ENV=development`), self-signed certs are accepted. In `production` mode, strict certificate validation is enforced.

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 16+
- RSA key pair
- SMTP credentials (for email verification)

### Local Setup

```bash
# 1. Clone
git clone <repo-url>
cd mein-idaas

# 2. Install dependencies
go mod download

# 3. Create .env file with required variables (see Configuration section above)

# 4. Create PostgreSQL database
psql -U postgres -c "CREATE DATABASE idaas;"

# 5. Run
go run main.go
```

Server starts on `http://localhost:4000`.

### Docker Setup

```bash
# Create .env file, then:
docker-compose up --build
```

This starts both PostgreSQL 16 and the application. Database schema is auto-migrated and default roles are seeded on startup.

### Verify

```bash
curl http://localhost:4000/health
# {"status":"ok"}
```

Swagger UI: `http://localhost:4000/swagger/`

## API Endpoints

All endpoints are prefixed with `/api/v1` unless noted otherwise.

### Health
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |

### Authentication
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/auth/register` | None | Register new user |
| `POST` | `/auth/verify` | None | Verify email with OTP |
| `POST` | `/auth/resend` | None | Resend verification OTP |
| `POST` | `/auth/login` | None | Login, returns token pair |
| `POST` | `/auth/refresh` | None | Rotate refresh token |

### MFA
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/auth/mfa/setup` | None | Generate TOTP secret + QR URL |
| `GET` | `/auth/mfa/qrcode` | None | Get QR code as PNG image |
| `GET` | `/auth/mfa/qrcode/base64` | None | Get QR code as base64 JSON |
| `POST` | `/auth/mfa/confirm` | None | Verify TOTP and enable MFA |

### Password Management
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/auth/password-change/send-otp` | None | Send OTP to change password |
| `POST` | `/auth/password-change` | None | Change password (old pw + OTP) |
| `POST` | `/auth/forgot-password/send-otp` | None | Request password reset OTP |
| `POST` | `/auth/forgot-password/reset` | None | Reset password with OTP |

> **Note:** Authentication middleware is not currently applied to routes. All endpoints are publicly accessible. Token validation logic exists in `util/token_verify.go` and can be wired into a middleware for protected routes.

### Example: Register → Verify → Login

```bash
# 1. Register
curl -X POST http://localhost:4000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","password":"SecurePass123!"}'

# Response: {"id":"uuid","name":"Alice","email":"alice@example.com"}

# 2. Verify email (check inbox for 6-digit code)
curl -X POST http://localhost:4000/api/v1/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","code":"123456"}'

# Response: {"message":"email verified"}

# 3. Login
curl -X POST http://localhost:4000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"SecurePass123!"}'

# Response: {"access_token":"eyJ...","refresh_token":"eyJ...","expires_in":900}
```

## JWT Token Structure

### Access Token (RS256)
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "roles": ["user"],
  "iss": "mein-idaas",
  "aud": ["self-hosted-idaas"],
  "iat": 1703247200,
  "exp": 1703248100
}
```

- `sub` — User UUID
- `roles` — Role codes for RBAC
- `iss` — Issuer (configurable via `JWT_ISSUER`)
- `aud` — Audience
- `exp` — Expiration (configurable via `JWT_ACCESS_TTL`, default 15 min)

### Refresh Token (RS256)
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "jti": "refresh-token-uuid",
  "iss": "mein-idaas",
  "iat": 1703247200,
  "exp": 1703852000
}
```

The refresh token's `jti` (JWT ID) corresponds to the `RefreshToken.ID` stored in the database, enabling rotation tracking.

## Security Features

### Password Hashing
- **Argon2id** with configurable parameters (time, memory, threads, key length, salt length)
- Each hash stores its own parameters (backward compatible — changing global params doesn't break existing passwords)
- Constant-time comparison to prevent timing attacks

### Token Security
- **RS256 asymmetric signing** — private key signs, public key verifies
- **Refresh token rotation** — every refresh issues a new pair and marks the old token replaced
- **Reuse detection** — if a used token is presented after the grace period (10s), it's flagged as theft
- **Token family tracking** — `replaced_by_token_id` creates a linked list of rotated tokens
- **Daily cleanup** — expired tokens are purged from DB at noon daily

### Rate Limiting
- 10 requests/second per IP
- IPs exceeding the limit are banned for 10 minutes
- Custom in-memory storage with periodic cleanup goroutine
- Banned IPs get `403 Forbidden`; over-limit gets `429 Too Many Requests`

### Email Verification
- 6-digit random OTP codes
- 5-minute TTL, stored in-memory
- Code deleted after successful verification (prevents replay)
- Async email sending (non-blocking API response)

## Dependencies

| Package | Purpose |
|---------|---------|
| `gofiber/fiber/v2` | HTTP framework (fasthttp-based) |
| `gofiber/swagger` | Swagger UI integration |
| `golang-jwt/jwt/v5` | JWT signing (RS256) and parsing |
| `gorm.io/gorm` | ORM (auto-migration, query building) |
| `gorm.io/driver/postgres` | PostgreSQL driver for GORM |
| `golang.org/x/crypto` | Argon2id password hashing |
| `google/uuid` | UUID v4 generation |
| `pquerna/otp` | TOTP generation and QR code rendering |
| `joho/godotenv` | `.env` file loading |
| `gopkg.in/gomail.v2` | SMTP email sending |
| `go-playground/validator/v10` | Struct validation |
| `swaggo/swag` | OpenAPI spec generation from annotations |

## Troubleshooting

### "email not verified" on login
The user's email hasn't been verified. Tell them to check their inbox for the 6-digit code, or call `POST /auth/resend`.

### "refresh token reuse detected: account locked for security"
A refresh token was used after the grace period expired — possible token theft. The user must log in again with credentials.

### "invalid or expired verification code"
OTP is wrong or older than 5 minutes. Request a new one via `POST /auth/resend`.

### RSA key errors
The application loads keys from environment variables as PEM strings, not file paths. Ensure `RSA_PRIVATE_KEY` and `RSA_PUBLIC_KEY` contain the full PEM content including `-----BEGIN/END-----` markers. Use `\n` for newlines. Both PKCS#1 and PKCS#8 formats are supported.

### Database connection errors
Verify PostgreSQL is running and the `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` variables are correct.

## License

MIT License — see `LICENSE` file.
