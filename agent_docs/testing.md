# Testing

Tests run in containers to match the production environment.

### Running Tests

```bash
# Run all tests (Go + JS) in container
./scripts/test.sh

# Run with coverage report
./scripts/test.sh --coverage
```

### Go Tests
Unit tests are in `*_test.go` files alongside the source code:
- `internal/config/config_test.go` - Configuration loading tests
- `internal/models/card_test.go` - Grid position and bingo detection tests
- `internal/middleware/*_test.go` - CSRF, auth, security, compression, caching tests
- `internal/services/auth_test.go` - Password hashing and session token tests
- `internal/services/card_test.go` - Bingo counting and position finding tests
- `internal/handlers/health_test.go` - Health check endpoint tests
- `internal/handlers/context_test.go` - User context tests
- `internal/testutil/testutil.go` - Test helper functions

### Frontend Tests
JavaScript tests in `web/static/js/tests/runner.js`:
- Tests for utility functions (escapeHtml, truncateText, parseHash)
- Tests for bingo detection algorithm
- Tests for grid position calculations
- Tests for progress calculations

Tests use only Node.js built-in modules (no npm dependencies).
