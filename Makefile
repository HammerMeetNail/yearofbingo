.PHONY: local down build up logs test lint clean assets

# Run full local rebuild: down, build assets, build container, up in background
local: down assets build up
	@echo "Local environment running. Use 'make logs' to view output or 'make down' to stop."

# Build hashed assets locally (needed because ./web is volume-mounted)
assets:
	./scripts/build-assets.sh

# Stop and remove containers
down:
	podman compose down

# Build containers
build:
	podman compose build

# Start containers in background
up:
	podman compose up -d

# View logs (follow mode)
logs:
	podman compose logs -f

# Run all tests in container
test:
	./scripts/test.sh

# Run linter
lint:
	golangci-lint run

# Clean up everything including volumes
clean:
	podman compose down -v
