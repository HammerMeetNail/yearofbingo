package models

import (
	"time"

	"github.com/google/uuid"
)

type ApiTokenScope string

const (
	ScopeRead      ApiTokenScope = "read"
	ScopeWrite     ApiTokenScope = "write"
	ScopeReadWrite ApiTokenScope = "read_write"
)

type ApiToken struct {
	ID          uuid.UUID     `json:"id"`
	UserID      uuid.UUID     `json:"user_id"`
	Name        string        `json:"name"`
	TokenHash   string        `json:"-"` // Never expose hash in JSON
	TokenPrefix string        `json:"token_prefix"`
	Scope       ApiTokenScope `json:"scope"`
	ExpiresAt   *time.Time    `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time    `json:"last_used_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
}

type CreateApiTokenParams struct {
	UserID      uuid.UUID
	Name        string
	TokenHash   string
	TokenPrefix string
	Scope       ApiTokenScope
	ExpiresAt   *time.Time
}
