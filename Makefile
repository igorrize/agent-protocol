.PHONY: run dev build test vet fmt tidy gate

run:
	go run ./cmd/server

dev:
	go run ./cmd/dev

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w . && goimports -w .

tidy:
	go mod tidy

# Quality gate (PLAN §11): fmt -> vet -> test -> build
gate: fmt vet test build
	@echo "✓ gate passed"
