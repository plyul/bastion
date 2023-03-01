BASTION_VERSION := $(shell build/get_git_ref.sh -b)
BASTION_COMPONENTS := bastion-server bastion-proxy
BASTION_SRC := $(shell find . -name "*.go")

GO := go
GO_BUILD := $(GO) build -v -a -ldflags "-X main.Version=$(BASTION_VERSION)"
GOFMT_LOG := fmt.log

DOCKER_EXE := $(shell which docker)
DOCKER_LINT_IMAGE := "golangci/golangci-lint"
DOCKER_BUILD_IMAGE := "golang:1.20"
DOCKER_GOPATH := /go
DOCKER_BASTION_SRC := $(DOCKER_GOPATH)/src/bastion

.PHONY: default
default:
	@echo Nice to meet you, engineer!

.PHONY: fmt
fmt:
	@gofmt -s -w $(BASTION_SRC)

.PHONY: lint
lint:
	@gofmt -s -l $(BASTION_SRC) > $(GOFMT_LOG)
	@[ ! -s $(GOFMT_LOG) ] || (echo "\ngofmt check failure, run 'make fmt'\n" | cat - $(GOFMT_LOG) && rm $(GOFMT_LOG) && false)
	@rm $(GOFMT_LOG)
	@$(DOCKER_EXE) run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.51.2 golangci-lint run \
		-E bodyclose -E dupl -E depguard -E gochecknoinits -E goconst -E gocritic -E revive -E govet -E prealloc -E unconvert -E unparam

.PHONY: build
build: $(BASTION_COMPONENTS)

$(BASTION_COMPONENTS): lint
	@echo Building $@ version $(BASTION_VERSION)
	@$(DOCKER_EXE) run --rm -v $(PWD):$(DOCKER_BASTION_SRC) -w $(DOCKER_BASTION_SRC) -e CGO_ENABLED=0 -e GOOS=linux $(DOCKER_BUILD_IMAGE) $(GO_BUILD) -o $@ cmd/$@/$@.go

.PHONY: clean
clean:
	@-rm -f $(BASTION_COMPONENTS)
