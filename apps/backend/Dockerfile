# syntax=docker/dockerfile:1
# Build context is the repo root — this stage builds apps/frontend, the next
# stage copies its output into the Go module's embed source before building.

FROM oven/bun:1 AS frontend-build
WORKDIR /src
COPY apps/frontend/package.json apps/frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY apps/frontend/ .
RUN bun run build

FROM golang:1.26 AS build
WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download
COPY apps/backend/ .
COPY --from=frontend-build /src/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -o /out/routerlens ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/routerlens /routerlens
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/routerlens"]
