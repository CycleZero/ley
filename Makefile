GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	Git_Bash="/c/Program Files/Git/bin/bash.exe"
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "cd `pwd` && find app -name *.proto -not -path '*/gateway/*'")
	CONFIG_PROTO_FILES=$(shell $(Git_Bash) -c "cd `pwd` && find conf -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "cd `pwd` && find api -name *.proto")
	INTERNAL_CONFIG_PROTO_FILES=$(shell $(Git_Bash) -c "cd `pwd` && find app -path '*/internal/conf/*.proto' -not -path '*/gateway/*'")
else
	INTERNAL_PROTO_FILES=$(shell find app -name *.proto -not -path '*/gateway/*')
	API_PROTO_FILES=$(shell find api -name *.proto)
	CONFIG_PROTO_FILES=$(shell find conf -name *.proto)
	INTERNAL_CONFIG_PROTO_FILES=$(shell find app -path '*/internal/conf/*.proto' -not -path '*/gateway/*')
endif


echo:
	@echo "========== Makefile Variables =========="
	@echo "GOHOSTOS: $(GOHOSTOS)"
	@echo "GOPATH: $(GOPATH)"
	@echo "VERSION: $(VERSION)"
	@echo "Git_Bash: $(Git_Bash)"
	@echo "INTERNAL_PROTO_FILES: $(INTERNAL_PROTO_FILES)"
	@echo "API_PROTO_FILES: $(API_PROTO_FILES)"
	@echo "CONFIG_PROTO_FILES: $(CONFIG_PROTO_FILES)"
	@echo "INTERNAL_CONFIG_PROTO_FILES: $(INTERNAL_CONFIG_PROTO_FILES)"
	@echo "=========================================="


.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/envoyproxy/protoc-gen-validate@latest

.PHONY: internal_proto
# generate internal proto
internal_proto:
	protoc --proto_path=./app \
  			--proto_path=./ \
		   --proto_path=./third_party \
		   --proto_path=./conf \
		   --go_out=paths=source_relative:./app \
		   $(INTERNAL_PROTO_FILES)

.PHONY: config
config:
	protoc --proto_path=./conf \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./conf \
	       $(CONFIG_PROTO_FILES)

	protoc --proto_path=./app \
			--proto_path=./ \
	       --proto_path=./third_party \
	       --proto_path=./conf \
 	       --go_out=paths=source_relative:./app \
	       $(INTERNAL_CONFIG_PROTO_FILES)




.PHONY: api
# generate api proto
api:
	protoc --proto_path=. \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: build
# build auth + blog services
build:
	mkdir -p bin/ && go build -o ./bin/auth ./app/auth/cmd/auth && go build -o ./bin/blog ./app/blog/cmd/blog

.PHONY: build-auth
# build auth service only
build-auth:
	mkdir -p bin/ && go build -o ./bin/auth ./app/auth/cmd/auth

.PHONY: build-blog
# build blog service only
build-blog:
	mkdir -p bin/ && go build -o ./bin/blog ./app/blog/cmd/blog

.PHONY: build-gateway
# build gateway service (separate module)
build-gateway:
	cd app/gateway && go build -o ../../bin/gateway ./cmd/gateway

.PHONY: build-all
# build all services including gateway
build-all: build build-gateway
.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: all
# generate all
all:
	make api
	make config
	make generate


wire:
	wire gen $(shell find ./app -name wire.go -not -path "*/test/*" -not -path "*/gateway/*" | xargs -n1 dirname | sort -u)

wire-all: wire
	cd app/gateway && wire gen ./cmd/gateway 2>/dev/null || true

rebuild: api config internal_proto wire build




# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
# run unit tests
test-unit:
	go test -v -short -count=1 -coverprofile=coverage.out ./app/... ./pkg/...

.PHONY: test-integration
# run integration tests (requires docker infrastructure)
test-integration:
	go test -v -count=1 -tags=integration ./app/... ./pkg/...

.PHONY: test-coverage
# open coverage report in browser
test-coverage:
	go tool cover -html=coverage.out

.PHONY: docker-build
# build all docker images
docker-build:
	docker-compose build

.PHONY: docker-up
# start all services via docker-compose
docker-up:
	docker-compose up -d

.PHONY: docker-down
# stop all services
docker-down:
	docker-compose down

.DEFAULT_GOAL := help
