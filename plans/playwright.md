# Playwright Local E2E Plan

## Goals
- Provide a single `make e2e` command that starts from a clean DB, seeds data, and runs Playwright UI tests in a container.
- Validate core front-end flows while asserting back-end behavior via the UI (and targeted API checks when needed).
- Provide separate commands for backend unit tests, frontend unit tests, and e2e tests.

## Decisions (from user)
- `make e2e` always starts fresh by dropping volumes (`podman compose down -v`).
- Playwright runs in a container; no npm install on the host.
- Default browser is Firefox; Chromium and WebKit are optional via flags.
- E2E tests use both seeded data and UI-created data.

## Proposed Workflow (`make e2e`)
1. `podman compose down -v` (destructive reset).
2. Build web assets (`./scripts/build-assets.sh`).
3. `podman compose up -d --build`.
4. Wait for `http://localhost:8080/health` to return 200.
5. Seed via `./scripts/seed.sh http://localhost:8080`.
6. Run Playwright in a container with `PLAYWRIGHT_BASE_URL` pointed at the app.
7. Leave the stack running for inspection (no cleanup by default).

## Playwright Container Strategy
- Add `Containerfile.playwright` based on `mcr.microsoft.com/playwright:<version>` (includes Firefox/Chromium/WebKit).
- Add a minimal `e2e/package.json` with `@playwright/test` installed into the image.
- Use `NODE_PATH` so tests can resolve `@playwright/test` without a host `node_modules`.
- Run Playwright on the compose network and set:
  - Seed script base URL: `http://localhost:8080` (host).
  - Playwright base URL: `http://app:8080` (container network).
- Add a compose profile/service `playwright` to simplify networking (`podman compose --profile e2e run --rm playwright`).

## Browser Selection
- `playwright.config.js` defines projects: `firefox` (default), `chromium`, `webkit`.
- `make e2e` runs `firefox` only.
- Allow overrides via `BROWSERS` env:
  - `make e2e BROWSERS=chromium`
  - `make e2e BROWSERS=firefox,webkit`
- Map `BROWSERS` to `--project` flags in `scripts/e2e.sh`.

## Test Coverage Outline
## Current E2E Scenarios
- `tests/e2e/seeded-data.spec.js`
  - Seeded dashboard cards render with progress and FREE space.
  - Friend reactions appear on seeded friend cards.
  - Cards can be archived via bulk actions and show archived view.
- `tests/e2e/anonymous-flow.spec.js`
  - Anonymous draft creation, shuffle (FREE stays), reload persistence.
  - Anonymous finalize triggers auth flow and imports card.
- `tests/e2e/anonymous-alternatives.spec.js`
  - Anonymous finalize supports login instead of register.
  - Anonymous card conflicts allow keeping the existing card (discarding anon card).
- `tests/e2e/authenticated-flow.spec.js`
  - Logged-in user configures header/FREE, finalizes, completes/uncompletes.
  - Clone flow supports grid size/header changes on new draft.
- `tests/e2e/bulk-actions.spec.js`
  - Bulk visibility updates, export ZIP download, and bulk delete.
- `tests/e2e/editor-actions.spec.js`
  - Draft goal edit/remove updates grid and progress.
  - Finalized visibility toggle switches between visible/private.
- `tests/e2e/auth-flows.spec.js`
  - Magic link login via email token.
  - Password reset via emailed token.
  - Email verification banner + resend + verify flow.
- `tests/e2e/friend-management.spec.js`
  - Cancel sent friend requests, reject incoming requests.
  - Remove an existing friend.
- `tests/e2e/profile-settings.spec.js`
  - Searchable toggle gates friend search visibility.
  - Password change validation and re-login with new password.
- `tests/e2e/profile-tokens.spec.js`
  - Create/revoke API tokens and copy-to-clipboard toast.
  - Revoke all tokens with confirmation.
- `tests/e2e/social-flow.spec.js`
  - Friend request, accept flow, view friend card, add reaction.

## Unit Test Separation
- Add make targets:
  - `make test-backend`: Go unit tests only.
  - `make test-frontend`: JS unit tests only.
  - `make test`: existing combined run.
- Implementation options:
  - Extend `./scripts/test.sh` with `--go` and `--js` flags, or
  - Add `./scripts/test-go.sh` and `./scripts/test-js.sh`.

## Files to Add/Modify
- `scripts/e2e.sh` (or `scripts/playwright.sh`)
- `Containerfile.playwright`
- `package.json` + `package-lock.json` (Playwright only)
- `playwright.config.js`
- `tests/e2e/*.spec.js`
- `Makefile` targets (`e2e`, `e2e-headed`, `test-backend`, `test-frontend`)
- Update docs in `agent_docs/testing.md`

## Acceptance Criteria
- `make e2e` always starts from a clean DB, seeds data, runs Playwright in Firefox, and returns non-zero on failure.
- `make e2e BROWSERS=chromium` and `make e2e BROWSERS=webkit` work.
- Backend-only and frontend-only unit test targets run independently.
- E2E suite covers both seeded and UI-created data flows.

## Scenario Backlog
- None (all current backlog items implemented).
