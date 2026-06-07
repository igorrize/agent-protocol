.PHONY: run orchestrate dev build test vet fmt tidy gate

run:
	go run ./cmd/agent-protocol serve

orchestrate:
	go run ./cmd/agent-protocol orchestrate

dev:
	go run ./cmd/dev

build:
	go build -o bin/agent-protocol ./cmd/agent-protocol

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w . && goimports -w .

tidy:
	go mod tidy

# Quality gate: format, vet, test, then build everything.
gate: fmt vet test
	go build ./...
	@echo "✓ gate passed"
