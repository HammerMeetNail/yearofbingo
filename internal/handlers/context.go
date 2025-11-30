package handlers

import (
	"context"

	"github.com/HammerMeetNail/yearofbingo/internal/models"
)

type contextKey string

const (
	userContextKey       contextKey = "user"
	tokenScopeContextKey contextKey = "token_scope"
)

func SetUserInContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func GetUserFromContext(ctx context.Context) *models.User {
	user, _ := ctx.Value(userContextKey).(*models.User)
	return user
}

func SetTokenScopeInContext(ctx context.Context, scope models.ApiTokenScope) context.Context {
	return context.WithValue(ctx, tokenScopeContextKey, scope)
}

func GetTokenScopeFromContext(ctx context.Context) models.ApiTokenScope {
	scope, _ := ctx.Value(tokenScopeContextKey).(models.ApiTokenScope)
	return scope
}
