.PHONY: dev build build-all test lint migrate create-admin tidy web web-build compose-up compose-down seed

# ---- backend (Go module `router-lens` lives in ./apps/backend) ----
dev:
	cd apps/backend && go run ./cmd/server

build:
	cd apps/backend && go build -o bin/routerlens ./cmd/server

build-all: web-build
	cd apps/backend && go build -o bin/routerlens ./cmd/server

test:
	cd apps/backend && go test -race -cover ./...

lint:
	cd apps/backend && golangci-lint run

tidy:
	cd apps/backend && go mod tidy

migrate:
	cd apps/backend && go run ./cmd/server -migrate-only

create-admin:
	@cd apps/backend && go run ./cmd/server -create-admin -email="$(EMAIL)" -password="$(PASSWORD)" -name="$(NAME)"

# ---- frontend (Vite + React lives in ./apps/frontend) ----
web:
	cd apps/frontend && bun install && bun run dev

web-build:
	cd apps/frontend && bun install && bun run build
	rm -rf apps/backend/internal/web/dist
	mkdir -p apps/backend/internal/web/dist
	cp -r apps/frontend/dist/. apps/backend/internal/web/dist/
	touch apps/backend/internal/web/dist/.gitkeep

# ---- docker (single deployable; build context = repo root) ----
compose-up:
	docker compose up --build

compose-down:
	docker compose down

# ---- optional demo data (run after first-run setup has created a user) ----
seed:
	psql "$$(grep DATABASE_URL apps/backend/.env | cut -d= -f2-)" -f scripts/seed.sql
