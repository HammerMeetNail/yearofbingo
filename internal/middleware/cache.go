package middleware

import (
	"net/http"
	"strings"
)

// CacheControl adds appropriate cache headers to responses.
type CacheControl struct{}

// NewCacheControl creates a new cache control middleware.
func NewCacheControl() *CacheControl {
	return &CacheControl{}
}

// Apply adds cache headers based on the request path.
func (c *CacheControl) Apply(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Set cache headers based on content type
		switch {
		case strings.HasPrefix(path, "/static/"):
			// Static assets - cache for 1 year (immutable)
			// In production, assets should have content hashes in filenames
			c.setStaticCacheHeaders(w, path)

		case strings.HasPrefix(path, "/api/"):
			// API responses should not be cached
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
			w.Header().Set("Pragma", "no-cache")

		case path == "/" || path == "":
			// HTML pages - cache briefly but revalidate
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")

		default:
			// Default - no caching
			w.Header().Set("Cache-Control", "no-store")
		}

		next.ServeHTTP(w, r)
	})
}

// setStaticCacheHeaders sets appropriate cache headers for static files.
func (c *CacheControl) setStaticCacheHeaders(w http.ResponseWriter, path string) {
	lowerPath := strings.ToLower(path)

	// Long-lived immutable assets (fonts, images)
	if isImmutableAsset(lowerPath) {
		// Cache for 1 year
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}

	// CSS and JS - cache for 1 day with revalidation
	// In development, we want faster refresh; in production with hashed filenames, use longer cache
	if strings.HasSuffix(lowerPath, ".css") || strings.HasSuffix(lowerPath, ".js") {
		w.Header().Set("Cache-Control", "public, max-age=86400, must-revalidate")
		return
	}

	// Default for other static files - 1 hour
	w.Header().Set("Cache-Control", "public, max-age=3600")
}

// isImmutableAsset returns true for assets that don't change.
func isImmutableAsset(path string) bool {
	immutableExtensions := []string{
		".woff", ".woff2", ".ttf", ".otf", ".eot",
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".ico", ".svg",
		".mp4", ".webm", ".mp3", ".ogg",
	}

	for _, ext := range immutableExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
