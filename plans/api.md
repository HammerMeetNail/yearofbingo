# API Token Access Plan

## Overview

Allow users to generate API tokens for programmatic access to their bingo cards without sharing passwords. Tokens have configurable permissions (read, write, or both) and expiration dates. Interactive API documentation available via OpenAPI/Swagger.

## User Stories

1. As a user, I want to generate an API token so I can automate bingo card management
2. As a user, I want to control what permissions my token has (read-only for sharing, write for automation)
3. As a user, I want to set when my token expires for security
4. As a user, I want to revoke tokens I no longer need
5. As a user, I want to see documentation with examples so I can use the API easily

## Design Decisions

- **Token format**: Opaque random tokens (32 bytes, base64 encoded) - simple, secure, no payload to decode
- **Storage**: Hash tokens before storing (like session tokens) for security
- **Scopes**: Simple model - `read`, `write`, or `read+write`
- **Expiration**: User-configurable, default 1 month, no maximum
- **Token limit**: Unlimited tokens per user
- **Documentation**: OpenAPI 3.0 spec with Swagger UI

## Database Schema

### New Table: `api_tokens`

```sql
CREATE TABLE api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,           -- User-friendly label (e.g., "Excel Import Script")
    token_hash VARCHAR(64) NOT NULL,      -- SHA-256 hash of the token
    token_prefix VARCHAR(8) NOT NULL,     -- First 8 chars for identification (e.g., "yob_abc1...")
    scope VARCHAR(20) NOT NULL,           -- 'read', 'write', 'read_write'
    expires_at TIMESTAMP WITH TIME ZONE,  -- NULL means never expires
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT valid_scope CHECK (scope IN ('read', 'write', 'read_write'))
);

CREATE INDEX idx_api_tokens_user_id ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_token_hash ON api_tokens(token_hash);
CREATE INDEX idx_api_tokens_prefix ON api_tokens(token_prefix);
```

### Migration

```
migrations/000X_api_tokens.up.sql
migrations/000X_api_tokens.down.sql
```

## API Endpoints

### Token Management (requires session auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/tokens` | List all tokens for current user (metadata only, not the token value) |
| POST | `/api/tokens` | Create a new token |
| DELETE | `/api/tokens/{id}` | Revoke a specific token |
| DELETE | `/api/tokens` | Revoke all tokens |

### Token Creation Request

```json
POST /api/tokens
{
    "name": "Excel Import Script",
    "scope": "read_write",
    "expires_in_days": 30
}
```

### Token Creation Response

```json
{
    "token": {
        "id": "uuid",
        "name": "Excel Import Script",
        "token": "yob_EXAMPLE_TOKEN_VALUE_HERE",  // Only shown once!
        "token_prefix": "yob_abc1",
        "scope": "read_write",
        "expires_at": "2025-12-30T00:00:00Z",
        "created_at": "2025-11-30T00:00:00Z"
    },
    "warning": "Save this token now. You won't be able to see it again."
}
```

### Token List Response

```json
{
    "tokens": [
        {
            "id": "uuid",
            "name": "Excel Import Script",
            "token_prefix": "yob_abc1",
            "scope": "read_write",
            "expires_at": "2025-12-30T00:00:00Z",
            "last_used_at": "2025-11-30T12:00:00Z",
            "created_at": "2025-11-30T00:00:00Z"
        }
    ]
}
```

## Token Authentication

### Header Format

```
Authorization: Bearer yob_abc123...xyz789
```

### Authenticated Endpoints

Tokens can access these endpoints based on scope:

| Endpoint | Read | Write |
|----------|------|-------|
| GET /api/auth/me | ✓ | ✓ |
| GET /api/cards | ✓ | ✓ |
| GET /api/cards/{id} | ✓ | ✓ |
| GET /api/cards/{id}/stats | ✓ | ✓ |
| POST /api/cards | | ✓ |
| POST /api/cards/{id}/items | | ✓ |
| PUT /api/cards/{id}/items/{pos} | | ✓ |
| DELETE /api/cards/{id}/items/{pos} | | ✓ |
| POST /api/cards/{id}/swap | | ✓ |
| POST /api/cards/{id}/shuffle | | ✓ |
| POST /api/cards/{id}/finalize | | ✓ |
| PUT /api/cards/{id}/items/{pos}/complete | | ✓ |
| PUT /api/cards/{id}/items/{pos}/uncomplete | | ✓ |
| PUT /api/cards/{id}/items/{pos}/notes | | ✓ |
| GET /api/suggestions | ✓ | ✓ |
| GET /api/suggestions/categories | ✓ | ✓ |

**Not accessible via token** (session-only):
- Friend management endpoints
- Reaction endpoints
- Token management endpoints
- Password/email changes
- Support requests

## Implementation

### Phase 1: Database & Models

1. Create migration for `api_tokens` table
2. Add `ApiToken` model in `internal/models/api_token.go`
3. Add `ApiTokenService` in `internal/services/api_token.go`
   - `Create(userID, name, scope, expiresInDays) (*ApiToken, plainToken, error)`
   - `List(userID) ([]ApiToken, error)`
   - `Delete(userID, tokenID) error`
   - `DeleteAll(userID) error`
   - `ValidateToken(plainToken) (*ApiToken, error)`
   - `UpdateLastUsed(tokenID) error`

### Phase 2: Token Management API

1. Add handlers in `internal/handlers/api_token.go`
   - `HandleListTokens`
   - `HandleCreateToken`
   - `HandleDeleteToken`
   - `HandleDeleteAllTokens`
2. Register routes in `cmd/server/main.go`
3. Token generation:
   - Generate 32 random bytes
   - Prefix with `yob_` for easy identification
   - Base64 encode for the user
   - SHA-256 hash for storage

### Phase 3: Token Authentication Middleware

1. Update `internal/middleware/auth.go`:
   - Check for `Authorization: Bearer` header first
   - If present, validate token and load user
   - If not, fall back to session cookie auth
   - Store token scope in request context
2. Add scope checking helper:
   - `RequireScope(scope string)` middleware wrapper
   - Returns 403 if token lacks required scope
3. Apply scope requirements to existing handlers

### Phase 4: OpenAPI Documentation

1. Create `web/static/openapi.yaml` with full API spec
2. Add Swagger UI page:
   - Download Swagger UI dist files to `web/static/swagger/`
   - Create handler to serve Swagger UI at `/api/docs`
   - Configure to load `openapi.yaml`
3. Add "API" link to navbar in `index.html`
4. Document all public endpoints with:
   - Request/response schemas
   - Authentication requirements
   - Scope requirements
   - Example requests/responses

### Phase 5: Token Management UI

1. Add "API Tokens" section to Profile page (`#profile`):
   - List existing tokens with name, prefix, scope, expiration, last used
   - "Create Token" button opens modal
   - Delete button for each token
   - "Revoke All" button with confirmation
2. Create Token modal:
   - Name input (required)
   - Scope dropdown (read, write, read+write)
   - Expiration dropdown (1 week, 1 month, 3 months, 6 months, 1 year, never)
   - Create button
3. Token display modal (after creation):
   - Show full token with copy button
   - Warning that token won't be shown again
   - Dismiss button

## Security Considerations

1. **Token storage**: Only store SHA-256 hash, never plaintext
2. **Token display**: Only show token once at creation time
3. **Rate limiting**: Consider stricter limits for token auth (future enhancement)
4. **Audit logging**: Log token usage with token prefix for debugging
5. **Scope enforcement**: Always check scope before allowing operations
6. **Expiration check**: Validate expiration on every request
7. **HTTPS only**: Tokens only transmitted over HTTPS
8. **No CSRF for token auth**: Bearer tokens don't need CSRF protection (not cookie-based)

## UI Mockups

### Profile Page - API Tokens Section

```
┌─────────────────────────────────────────────────────────────┐
│ API Tokens                                    [Create Token]│
├─────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Excel Import Script                              [Delete]│ │
│ │ yob_abc1... • read+write • Expires Dec 30, 2025         │ │
│ │ Last used: 2 hours ago                                  │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ Backup Script                                    [Delete]│ │
│ │ yob_xyz9... • read • Never expires                      │ │
│ │ Last used: Never                                        │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                             │
│ [Revoke All Tokens]                                         │
└─────────────────────────────────────────────────────────────┘
```

### Create Token Modal

```
┌─────────────────────────────────────────┐
│ Create API Token                      ✕ │
├─────────────────────────────────────────┤
│                                         │
│ Name                                    │
│ ┌─────────────────────────────────────┐ │
│ │ My Import Script                    │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ Permissions                             │
│ ┌─────────────────────────────────────┐ │
│ │ Read & Write                      ▼ │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ Expires                                 │
│ ┌─────────────────────────────────────┐ │
│ │ 1 month                           ▼ │ │
│ └─────────────────────────────────────┘ │
│                                         │
│              [Cancel] [Create Token]    │
└─────────────────────────────────────────┘
```

### Token Created Modal

```
┌─────────────────────────────────────────┐
│ Token Created                         ✕ │
├─────────────────────────────────────────┤
│                                         │
│ ⚠️ Copy this token now!                 │
│ You won't be able to see it again.      │
│                                         │
│ ┌─────────────────────────────────────┐ │
│ │ yob_a3Hk9mNpQr...xYz7WvB2         📋│ │
│ └─────────────────────────────────────┘ │
│                                         │
│ Use this token in the Authorization     │
│ header:                                 │
│                                         │
│ Authorization: Bearer yob_a3Hk...       │
│                                         │
│                              [Done]     │
└─────────────────────────────────────────┘
```

### Navbar with API Link

```
┌─────────────────────────────────────────────────────────────┐
│ 🎯 Year of Bingo          [FAQ] [API] [Profile ▼] [Logout] │
└─────────────────────────────────────────────────────────────┘
```

## API Documentation Content

The OpenAPI spec should document:

1. **Authentication**
   - Session-based (cookies) for web UI
   - Token-based (Bearer) for API access
   - How to get a token

2. **Common Use Cases** with examples:
   - Import a bingo card from spreadsheet
   - Export card data
   - Mark items as complete
   - Get card statistics

3. **Error Responses**
   - 401 Unauthorized (invalid/expired token)
   - 403 Forbidden (insufficient scope)
   - 404 Not Found
   - 422 Validation Error

4. **Rate Limits** (if implemented)

## Testing

1. **Unit tests**:
   - Token generation and hashing
   - Scope validation
   - Expiration checking

2. **Integration tests**:
   - Create token via API
   - Use token to read cards
   - Use token to write cards
   - Verify scope restrictions
   - Verify expiration handling

3. **Manual testing**:
   - Full flow: create token → use in curl → revoke
   - Swagger UI try-it-out functionality

## Rollout

1. Deploy database migration
2. Deploy backend changes (token management + auth middleware)
3. Deploy frontend changes (profile UI + navbar)
4. Deploy Swagger UI and documentation
5. Announce feature to users

## Future Enhancements

- Token usage analytics (requests per day/week)
- Webhook notifications on card changes
- OAuth2 for third-party app integrations
- More granular scopes if needed
- IP allowlisting for tokens

---

## Implementation Status: COMPLETED

### Phase 1: Database & Models ✅
- Created migration `000010_api_tokens.up.sql`
- Created `ApiToken` model and service

### Phase 2: Token Management API ✅
- Implemented list, create, delete, delete all endpoints

### Phase 3: Token Authentication Middleware ✅
- Implemented `AuthMiddleware` updates for Bearer token
- Added `RequireScope` and `RequireSession` helpers
- Secured all endpoints in `main.go`

### Phase 4: OpenAPI Documentation ✅
- Created `openapi.yaml`
- Added Swagger UI at `/api/docs`
- Added API link to navbar

### Phase 5: Token Management UI ✅
- Added API Tokens section to Profile page
- Added token creation and revocation UI