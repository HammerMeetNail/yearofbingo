# Testing

Tests run in containers to match the production environment.

### Running Tests

```bash
# Run all tests (Go + JS) in container
./scripts/test.sh

# Run Go tests only
./scripts/test.sh --go
make test-backend

# Run JS tests only
./scripts/test.sh --js
make test-frontend

# Run with coverage report
./scripts/test.sh --coverage

# Run Go tests with a coverage summary (writes coverage.out)
make coverage
```

### E2E Tests (Playwright)

```bash
# Destructive: resets DB volumes, seeds data, runs Playwright in Firefox
make e2e

# Run in headed mode
make e2e-headed

# Override browsers
make e2e BROWSERS=chromium
make e2e BROWSERS=firefox,webkit

# Full desktop matrix plus Android/iPhone emulation
make e2e BROWSERS=firefox,chromium,webkit,mobile-chromium,mobile-webkit

# Focused mobile touch/layout coverage
BROWSERS=mobile-chromium,mobile-webkit ./scripts/e2e.sh mobile-flows.spec.js
```

Artifacts are written to `test-results` and `playwright-report`.
Set `E2E_RESULTS_DIR` and `E2E_REPORT_DIR` to absolute host paths to keep local E2E artifacts outside the repository.

Notes:
- `make e2e` explicitly sets `FEATURE_AI_ENABLED=true` with `AI_STUB=1` so AI wizard flows are deterministic without external APIs.
- For the default disabled state, run `FEATURE_AI_ENABLED=false ./scripts/e2e.sh ai-disabled.spec.js`.
- Test the disabled state across devices with `FEATURE_AI_ENABLED=false BROWSERS=firefox,chromium,webkit,mobile-chromium,mobile-webkit ./scripts/e2e.sh ai-disabled.spec.js mobile-flows.spec.js`.
- Outside E2E, AI defaults off. Real AI requires `FEATURE_AI_ENABLED=true` plus a configured provider; `AI_STUB` cannot bypass the switch.

Desktop projects run the full suite. Mobile projects run the anonymous/authenticated card, AI wizard/guide, disabled-AI, and mobile flow specs using Pixel 7 (Chromium) and iPhone 13 (WebKit) emulation. Mobile flows check viewport overflow, menu navigation, editing, completion, feature gating, and drag cancellation/persistence. Taps use native touchscreen input in both engines. Chromium long presses use native CDP touch input; WebKit long presses use synthetic DOM touch events to exercise the handlers. These tests do not replace physical-device testing.

The CI E2E matrix runs each project separately and repeats the disabled-AI/mobile flow specs with AI off. Existing CI trigger rules still apply. Test identities include project, feature-switch mode, repeat, and retry so the two passes can share a seeded database. Each run writes a `results.json` summary alongside its raw test artifacts.

Containers use the `app.test` network alias for the browser URL. WebKit rejects the Strict cookies on the single-label `app` hostname; the reserved `.test` hostname lets every engine exercise the same CSRF/session cookie policy. Keep `PLAYWRIGHT_BASE_URL`, `APP_BASE_URL`, and OAuth callback URLs aligned when overriding this address.

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
