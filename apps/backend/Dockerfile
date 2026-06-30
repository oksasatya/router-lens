# syntax=docker/dockerfile:1
# Build context is the repo root (so a later stage can also build apps/frontend
# and embed it). Paths below are relative to the root context.
FROM golang:1.26 AS build
WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download
COPY apps/backend/ .
RUN CGO_ENABLED=0 go build -o /out/routerlens ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/routerlens /routerlens
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/routerlens"]
