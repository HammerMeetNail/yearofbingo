CREATE TABLE ai_generation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    model VARCHAR(50) NOT NULL,
    tokens_input INT NOT NULL,
    tokens_output INT NOT NULL,
    duration_ms INT NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ai_logs_user_date ON ai_generation_logs(user_id, created_at);
