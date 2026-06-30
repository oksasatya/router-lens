.PHONY: dev build test lint migrate create-admin tidy web web-build compose-up compose-down

# ---- backend (Go module `router-lens` lives in ./apps/backend) ----
dev:
	cd apps/backend && go run ./cmd/server

build:
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
	cd apps/backend && go run ./cmd/server -create-admin

# ---- frontend (TanStack Start lives in ./apps/frontend) ----
web:
	cd apps/frontend && pnpm dev

web-build:
	cd apps/frontend && pnpm build

# ---- docker (single deployable; build context = repo root) ----
compose-up:
	docker compose up --build

compose-down:
	docker compose down
