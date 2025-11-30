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
