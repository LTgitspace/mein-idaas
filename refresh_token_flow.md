# Refresh Token Flow — Step by Step

This document describes the exact token lifecycle in Mein IDaaS, matching the actual code implementation.

## Key Concepts

- **Access Token**: Short-lived RS256 JWT (default 15 min). Stateless — no DB lookup needed.
- **Refresh Token**: Long-lived RS256 JWT (default 7 days). Stored hashed in DB for rotation tracking.
- **Token Rotation**: Every refresh replaces the old token pair. Old refresh token is marked as replaced.
- **Grace Period**: 10-second window where a just-replaced token can still be safely retried (handles network race conditions).
- **Theft Detection**: If a used token is presented after the grace period, the system locks down and rejects it.

## 1. Initial Login

```
User sends: POST /api/v1/auth/login
{
  "email": "alice@example.com",
  "password": "mypassword"
}

Server does:
1. Look up user by email
2. Find password credential (type: "password")
3. Verify password with Argon2id comparison
4. Check is_email_verified (403 + resend OTP if false)
5. Generate token pair with RS256:
   - Access token: contains sub (user UUID), roles, exp (+15 min)
   - Refresh token: contains sub, jti (token UUID), exp (+7 days)
6. Hash refresh token string with SHA-256
7. Store in refresh_tokens table:
   - id = jti from token
   - user_id = user's UUID
   - token_hash = SHA-256(token_string)
   - expires_at = now + 7 days
   - client_ip, user_agent (for audit)

Server responds:
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJSUzI1NiIs...",
  "expires_in": 900  // 15 minutes in seconds
}

Both tokens are JWTs signed with the server's RSA private key.
```

## 2. Using the Access Token

```
Time: 0:00 - 14:59 (within 15 minutes)

Client sends: GET /api/protected-resource
Header: Authorization: Bearer eyJhbGciOiJSUzI1NiIs...

Server does:
1. Parse and validate JWT signature using RSA public key
2. Check exp claim (not expired)
3. Extract sub (user ID) and roles from claims
4. ✅ Token valid → process request

Response: 200 OK
```

## 3. Access Token Expired → Refresh

```
Time: 15:01 (access token expired)

Client sends: GET /api/protected-resource
Header: Authorization: Bearer eyJ...(expired)...

Server does:
1. Parse JWT → ❌ expired
2. Return 401 Unauthorized

Client catches 401 → knows access token expired
Client automatically sends refresh request
```

## 4. Refresh Token Rotation (Normal — First Use)

```
Client sends: POST /api/v1/auth/refresh
{
  "refresh_token": "eyJhbGciOiJSUzI1NiIs..."  (the current refresh token JWT)
}

Server does:
1. Parse refresh token JWT → extract:
   - sub: user UUID
   - jti: token UUID (matched to DB id)

2. Load token from DB: SELECT * FROM refresh_tokens WHERE id = jti

3. Security checks:
   - Token exists in DB? ✅
   - user_id matches sub from JWT? ✅
   - revoked_at IS NULL? ✅ (not revoked)
   - expires_at > NOW()? ✅ (not expired)
   - replaced_at IS NULL? ✅ (first time using this token)

4. All checks pass → NORMAL ROTATION:
   a. Generate NEW access token (15 min) + NEW refresh token (7 days)
   b. Save NEW refresh token to DB (new id, new hash, new expires_at)
   c. Mark OLD token as replaced:
      - replaced_at = NOW()
      - replaced_by_token_id = new token's id
   d. Old token is NOT revoked — it's marked as "replaced" for grace period handling

5. Return new pair:
{
  "access_token": "eyJ...(NEW)...",
  "refresh_token": "eyJ...(NEW)...",
  "expires_in": 900
}

Client replaces stored tokens with the new ones.
```

## 5. Grace Period Retry (Concurrent Request Safety)

```
Scenario: Two API calls happen at nearly the same time, both detect
the access token is expired, and both try to refresh.

Request A arrives first:
├── Server processes normally (see step 4)
├── Issues Token B (new pair)
├── Marks Token A as replaced (replaced_at = NOW())

Request B arrives ~50ms later (still using Token A):
├── Server parses Token A, loads from DB
├── Sees: replaced_at = NOW() (set by Request A)
├── Calculates: NOW() - replaced_at = 50ms
├── Grace period = 10 seconds (configurable)
├── 50ms < 10s → WITHIN GRACE PERIOD ✅
├── Safe retry — not a theft attempt, just a race condition

Server handles grace period retry:
├── Finds the child token via replaced_by_token_id → Token B
├── Generates ONLY a new access token (reuses existing refresh Token B)
├── Re-signs Token B's ID as a JWT
└── Returns:
    {
      "access_token": "eyJ...(fresh access)...",
      "refresh_token": "eyJ...(re-signed Token B)...",
      "expires_in": 900
    }
```

## 6. Theft Detection (Replay Attack)

```
Scenario: An attacker steals Token A and tries to use it after
the legitimate user has already rotated it.

Legitimate user refreshes Token A → gets Token B
├── Token A is marked: replaced_at = <timestamp>

15 seconds later, attacker sends Token A:
├── Server loads Token A from DB
├── Sees: replaced_at IS SET
├── Calculates: NOW() - replaced_at = 15 seconds
├── Grace period = 10 seconds
├── 15s > 10s → OUTSIDE GRACE PERIOD ❌
├── THIS IS THEFT!

Server responds: 401 Unauthorized
{
  "error": "refresh token reuse detected: account locked for security"
}

Attacker is blocked. Legitimate user's Token B continues working.
```

## Timeline Visualization

```
LOGIN
│
├─ 0:00 ───────────────────────── 15:00 ────── 15:01
│  Access Token Valid              Expires       ❌ Expired
│  Refresh Token Valid (7 days)    Valid         Valid
│
└─ 15:01 → POST /auth/refresh with refresh token
   ├─ ✅ Validated in DB
   ├─ Generate NEW access token (reset to 0:00)
   ├─ Generate NEW refresh token (7 days from now)
   ├─ Mark OLD token as "replaced" (not revoked)
   └─ Return both new tokens

AFTER REFRESH
│
├─ 15:01 ──────────────────────── 30:01 ────── 30:02
│  NEW Access Token Valid          Expires       ❌ Expired
│  NEW Refresh Token (7 days)      Valid         Valid
│
└─ If access expires again at 30:02:
   └─ Repeat: send refresh token → get even newer tokens
   └─ OLD refresh token (from 15:01) marked "replaced", not revoked

GRACE PERIOD DETAIL
│
├─ Token A used for refresh at T=0
│  └─ Token A.replaced_at = T=0, Token A.replaced_by_token_id = Token B
│
├─ T=0 to T=10s: Grace Period Window
│  └─ Token A replayed → safe retry, returns Token B
│
├─ T=10s+: Outside Grace Period
│  └─ Token A replayed → THEFT DETECTED, rejected
```

## Database State During Rotation

```
refresh_tokens table BEFORE first refresh:
┌──────────┬──────────┬──────────────┬──────────────┬─────────────┬──────────────────────┬────────────┐
│ id       │ user_id  │ token_hash   │ expires_at   │ replaced_at │ replaced_by_token_id │ revoked_at │
├──────────┼──────────┼──────────────┼──────────────┼─────────────┼──────────────────────┼────────────┤
│ token-A  │ alice    │ hash(A_jwt)  │ +7 days      │ NULL        │ NULL                 │ NULL       │ ✅ Active
└──────────┴──────────┴──────────────┴──────────────┴─────────────┴──────────────────────┴────────────┘

AFTER first refresh (Token A → Token B):
┌──────────┬──────────┬──────────────┬──────────────┬─────────────┬──────────────────────┬────────────┐
│ id       │ user_id  │ token_hash   │ expires_at   │ replaced_at │ replaced_by_token_id │ revoked_at │
├──────────┼──────────┼──────────────┼──────────────┼─────────────┼──────────────────────┼────────────┤
│ token-A  │ alice    │ hash(A_jwt)  │ +7 days      │ NOW()       │ token-B              │ NULL       │ 🔄 Replaced
│ token-B  │ alice    │ hash(B_jwt)  │ +7 days      │ NULL        │ NULL                 │ NULL       │ ✅ Active
└──────────┴──────────┴──────────────┴──────────────┴─────────────┴──────────────────────┴────────────┘

AFTER second refresh (Token B → Token C):
┌──────────┬──────────┬──────────────┬──────────────┬─────────────┬──────────────────────┬────────────┐
│ id       │ user_id  │ token_hash   │ expires_at   │ replaced_at │ replaced_by_token_id │ revoked_at │
├──────────┼──────────┼──────────────┼──────────────┼─────────────┼──────────────────────┼────────────┤
│ token-A  │ alice    │ hash(A_jwt)  │ +7 days      │ NOW()-7d    │ token-B              │ NULL       │ 🔄 Replaced
│ token-B  │ alice    │ hash(B_jwt)  │ +7 days      │ NOW()       │ token-C              │ NULL       │ 🔄 Replaced
│ token-C  │ alice    │ hash(C_jwt)  │ +7 days      │ NULL        │ NULL                 │ NULL       │ ✅ Active
└──────────┴──────────┴──────────────┴──────────────┴─────────────┴──────────────────────┴────────────┘

Note: replaced tokens form a linked list (A → B → C).
This enables the grace period to walk the chain if needed.
```

## Daily Cleanup

A background goroutine runs daily at noon and deletes tokens where `expires_at < NOW()`. This keeps the `refresh_tokens` table from growing unbounded. Expired + replaced tokens are both cleaned up.

## Key Implementation Details

| Detail | Implementation |
|---|---|
| **Signing algorithm** | RS256 (RSA PKCS#1 v1.5 with SHA-256) |
| **Key storage** | RSA private/public key pair loaded from env vars as PEM strings |
| **Token hash** | SHA-256 of the full JWT string |
| **Token family** | `replaced_by_token_id` links parent → child |
| **Grace period** | Configurable via `REFRESH_GRACE_PERIOD` (default 10s) |
| **Daily cleanup** | Cron-style goroutine, runs at 12:00 PM server time |
| **Access TTL** | Configurable via `JWT_ACCESS_TTL` (default 15m) |
| **Refresh TTL** | Configurable via `JWT_REFRESH_TTL` (default 168h / 7 days) |

## Security Properties

- **One-time use**: Each refresh token is usable exactly once. After rotation, it's marked as replaced.
- **Grace period**: Protects against false positives from concurrent API calls.
- **Theft detection**: Reuse after grace period is treated as a stolen token and rejected.
- **No shared secret**: RS256 uses asymmetric keys — the private key never leaves the server.
- **Audit trail**: Every token records `client_ip` and `user_agent` at creation time.
