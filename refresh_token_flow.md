# Refresh Token Flow - Step by Step

## Initial Login

```
User sends: POST /auth/login
{
  "email": "alice@example.com",
  "password": "mypassword"
}

Server does:
1. Check if user exists and password is correct
2. Create ACCESS TOKEN
   - Valid for: 15 minutes
   - Contains: user_id, email
   - Type: JWT (no DB needed to verify)
   
3. Create REFRESH TOKEN
   - Valid for: 7 days
   - Is: a random 32-character string
   - Store in DB with user_id and expiry date
   - Hash it before storing (like passwords)

Server responds:
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "expires_in": 900  // 15 minutes in seconds
}

Client stores:
- access_token in memory (or secure storage)
- refresh_token in secure storage (httpOnly cookie or local storage)
```

## Using Access Token (Normal Requests)

```
Time: 0:00 - 14:59 (Within 15 minutes)

User makes: GET /api/profile
Header: Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

Server does:
1. Decode JWT (no DB lookup needed)
2. Check if valid and not expired
3. ✅ Token is valid → Return user profile

Response: 200 OK with user data
```

## Access Token Expires - Need to Refresh

```
Time: 15:01 (Access token expired)

User makes: GET /api/profile
Header: Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

Server does:
1. Try to decode JWT
2. ❌ Token is expired
3. Return 401 Unauthorized

Client catches 401 and knows: "My access token died, use refresh token"
```

## Using Refresh Token to Get New Access Token

```
Time: 15:01

Client automatically sends: POST /auth/refresh
{
  "refresh_token": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
}

Server does:
1. Look up refresh_token in DATABASE
   - Query: SELECT * FROM refresh_tokens WHERE token_hash = hash("a1b2c3...")
   
2. Check conditions:
   - Does it exist? ✅ Yes
   - Is it expired? ❌ No (expires in 6 days, 23 hours 59 min)
   - Is it revoked? ❌ No (revoked_at is NULL)
   
3. All checks pass! Generate NEW tokens:
   
   NEW ACCESS TOKEN:
   - Valid for: 15 minutes
   - Contains: user_id, email
   
   NEW REFRESH TOKEN:
   - Valid for: 7 days
   - Store in DB with same user_id
   
4. IMPORTANT: Revoke OLD refresh token
   - Find old token in DB
   - Set revoked_at = NOW()
   - Update in DB
   - Old token can never be used again
   
5. Return new tokens

Server responds:
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",  // NEW
  "refresh_token": "x9y8z7w6v5u4t3s2r1q0p9o8",  // NEW
  "expires_in": 900
}

Client updates:
- access_token = new one
- refresh_token = new one
```

## Timeline Visualization

```
LOGIN
│
├─ 0:00 ─────────────────────── 15:00 ────── 15:01
│  Access Token Valid         Expires        ❌ Expired
│  Refresh Token (7 days)     Still valid    Still valid
│
└─ 15:01 → Send refresh_token
   ├─ ✅ Validated in DB
   ├─ Generate NEW access token (0:00 - 15:00 again)
   ├─ Generate NEW refresh token (7 days from now)
   ├─ Revoke OLD refresh token (can't use anymore)
   └─ Return both new tokens

AFTER REFRESH
│
├─ 15:01 ─────────────────────── 30:01 ────── 30:02
│  NEW Access Token Valid      Expires        ❌ Expired
│  NEW Refresh Token (7 days)  Still valid    Still valid
│
└─ If access expires again at 30:02:
   └─ Repeat: send NEW refresh token → get even newer tokens
   └─ OLD refresh token (from 15:01) is already revoked, can't use
```

## What Happens if Refresh Token is Stolen?

```
Attacker steals: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"

Scenario 1: Attacker uses it FIRST (before legitimate user)
├─ Attacker: POST /auth/refresh with stolen token
├─ Server: ✅ Valid, generates new tokens for attacker
├─ Server: Revokes the original token
└─ Legitimate user: Tries to refresh
    ├─ Sends: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
    ├─ Server: Looks up token in DB
    └─ ERROR: "This token is revoked" ❌

Scenario 2: Legitimate user uses it first
├─ User: POST /auth/refresh with token
├─ Server: ✅ Valid, generates NEW tokens
├─ Server: Revokes the original token
└─ Attacker: Tries to use stolen token
    ├─ Sends: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
    ├─ Server: Looks up token in DB
    └─ ERROR: "This token is revoked" ❌

In both cases: Token can only be used ONCE
```

## Database Storage

```
refresh_tokens table:
┌──────────────────┬─────────────────┬──────────────────┬──────────────────┬───────────────┐
│ id               │ user_id         │ token_hash       │ expires_at       │ revoked_at    │
├──────────────────┼─────────────────┼──────────────────┼──────────────────┼───────────────┤
│ uuid-1           │ user-alice      │ hash(a1b2c3...)  │ 2025-12-28 10:00 │ NULL          │ ✅ Active
│ uuid-2           │ user-alice      │ hash(x9y8z7...)  │ 2025-12-29 10:00 │ NULL          │ ✅ Active
│ uuid-3           │ user-alice      │ hash(old123...)  │ 2025-12-27 10:00 │ 2025-12-22... │ ❌ Revoked
└──────────────────┴─────────────────┴──────────────────┴──────────────────┴───────────────┘

When user refreshes:
1. Find token by hash
2. Check: not expired AND revoked_at IS NULL
3. If both true: it's valid
4. Update the old one: set revoked_at = NOW()
```

## Code Flow

```
USER LOGS IN
    ↓
POST /auth/login
    ↓
✅ Password correct
    ↓
Generate access_token (JWT, 15 min)
Generate refresh_token (random string, 7 days)
Save refresh_token to DB
    ↓
Return both tokens to client
    ↓
═════════════════════════════════════════════════════════════════

ACCESS TOKEN EXPIRES
    ↓
User tries GET /api/profile with old access_token
    ↓
❌ JWT decode fails (expired)
Server returns 401
    ↓
Client sees 401, uses refresh_token
    ↓
POST /auth/refresh with refresh_token
    ↓
Look up token in DB
Check: exists? expires_at > NOW? revoked_at IS NULL?
    ↓
✅ All checks pass
    ↓
Generate NEW access_token (JWT, 15 min)
Generate NEW refresh_token (random string, 7 days)
Mark OLD refresh_token as revoked (revoked_at = NOW())
Save NEW refresh_token to DB
    ↓
Return both new tokens
    ↓
Client updates tokens
    ↓
User can now use new access_token for next 15 minutes
```

## Key Points

1. **Access Token**: JWT, short-lived (15 min), stateless (no DB lookup)
2. **Refresh Token**: Random string, long-lived (7 days), stored in DB
3. **Rotation**: Every refresh creates a NEW refresh token, OLD one is revoked
4. **Security**: Old token can only be used ONCE. After that it's marked revoked.
5. **Expiration**: Refresh token expires naturally after 7 days. Old revoked tokens are cleaned up periodically.
6. **Client Storage**: 
   - Access token: short-term (memory)
   - Refresh token: long-term secure storage (httpOnly cookie)