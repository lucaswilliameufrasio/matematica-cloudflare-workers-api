SHELL := /bin/bash

.PHONY: dev test migrate seed lint fmt

dev:
	go run ./cmd/server

test:
	go test ./...

migrate:
	goose -dir ./db/migrations postgres "$(DATABASE_URL)" up

seed:
	@echo "Add seed logic here if desired"

lint:
	golangci-lint run || true

fmt:
	gofumpt -l -w . || true
