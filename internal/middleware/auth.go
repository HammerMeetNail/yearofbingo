package middleware

import (
	"net/http"
	"strings"

	"github.com/HammerMeetNail/yearofbingo/internal/handlers"
	"github.com/HammerMeetNail/yearofbingo/internal/models"
	"github.com/HammerMeetNail/yearofbingo/internal/services"
)

const sessionCookieName = "session_token"

type AuthMiddleware struct {
	authService     *services.AuthService
	userService     *services.UserService
	apiTokenService *services.ApiTokenService
}

func NewAuthMiddleware(authService *services.AuthService, userService *services.UserService, apiTokenService *services.ApiTokenService) *AuthMiddleware {
	return &AuthMiddleware{
		authService:     authService,
		userService:     userService,
		apiTokenService: apiTokenService,
	}
}

// Authenticate validates the session or API token and adds user to context if valid.
// Does not reject unauthenticated requests.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check for Bearer token
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := m.apiTokenService.ValidateToken(r.Context(), tokenStr)
			if err == nil {
				// Valid token, get user
				user, err := m.userService.GetByID(r.Context(), token.UserID)
				if err == nil {
					// Add user and scope to context
					ctx := handlers.SetUserInContext(r.Context(), user)
					ctx = handlers.SetTokenScopeInContext(ctx, token.Scope)

					// Update last used
					_ = m.apiTokenService.UpdateLastUsed(r.Context(), token.ID)

					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// 2. Check for session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.authService.ValidateSession(r.Context(), cookie.Value)
		if err != nil {
			// Invalid session, continue without user
			next.ServeHTTP(w, r)
			return
		}

		// Add user to context
		ctx := handlers.SetUserInContext(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth rejects unauthenticated requests with 401.
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := handlers.GetUserFromContext(r.Context())
		if user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"Authentication required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireScope rejects requests that don't meet the required scope.
// Session-authenticated users always have full access.
func (m *AuthMiddleware) RequireScope(requiredScope models.ApiTokenScope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := handlers.GetUserFromContext(r.Context())
			if user == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"Authentication required"}`))
				return
			}

			tokenScope := handlers.GetTokenScopeFromContext(r.Context())

			// Session auth (no token scope) has full access
			if tokenScope == "" {
				next.ServeHTTP(w, r)
				return
			}

			allowed := false
			if tokenScope == models.ScopeReadWrite {
				allowed = true
			} else if tokenScope == requiredScope {
				allowed = true
			} else if requiredScope == models.ScopeRead && tokenScope == models.ScopeWrite {
				// Write scope permits Read actions per policy
				allowed = true
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"Insufficient token scope"}`))
				return
			}

			next.ServeHTTP(w, r)

		})

	}

}

// RequireSession rejects requests authenticated via API token.

func (m *AuthMiddleware) RequireSession(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		tokenScope := handlers.GetTokenScopeFromContext(r.Context())

		if tokenScope != "" {

			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusForbidden)

			_, _ = w.Write([]byte(`{"error":"Token authentication not allowed for this endpoint"}`))

			return

		}

		next.ServeHTTP(w, r)

	})

}
