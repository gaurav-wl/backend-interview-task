# Variables
PROTO_DIR := proto
PROTO_FILES := $(wildcard $(PROTO_DIR)/*.proto)
MOCKERY := $(shell go env GOPATH)/bin/mockery

.PHONY: deps
deps:
	@echo "Installing necessary tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/vektra/mockery/v2@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/fullstorydev/grpcui/cmd/grpcui@latest

.PHONY: deps-ci
deps-ci:
	@echo "Installing necessary tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONY: proto
proto:
	@echo "Generating protobuf files..."
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative $(PROTO_FILES)

.PHONY: build
build: proto sqlc
	@echo "Building application..."
	go build -o bin/server ./cmd/server

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v -race ./...

.PHONY: sqlc
sqlc:
	sqlc generate

.PHONY: run
run: build
	@echo "Starting server..."
	./bin/server

mock:
	@echo ">> Generating mocks"
	$(MOCKERY) --all --config=mockery.yaml

.PHONY: generate-all
generate-all: mock sqlc proto
	@echo ">> Generating mocks, sqlc, and protobuf files"


.PHONY: docker-up
docker-up: sqlc proto
	@echo "Running server in docker..."
	docker compose up --build -d

.PHONY: test-ui
test-ui:
	grpcui -plaintext localhost:8080

