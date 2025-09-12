FROM golang:1.25 AS builder
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /out/app ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /out/app /app
COPY api/openapi.yaml /api/openapi.yaml
COPY db/migrations /db/migrations
ENV HTTP_ADDR=:8080
ENV AUTO_MIGRATE=true
EXPOSE 8080
ENTRYPOINT ["/app"]
