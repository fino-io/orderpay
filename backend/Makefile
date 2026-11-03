.PHONY: env test test-fast test-full test-race test-coverage mockgen vet lint sec vuln verify

GOPROXY := https://goproxy.cn,https://proxy.golang.org,direct
export GOPROXY
GO_MOCKGEN=$(shell command -v mockgen 2> /dev/null || echo '')
GO_MOCKGEN_URI=github.com/golang/mock/mockgen@v1.6.0
GO_LINT=$(shell command -v golangci-lint 2> /dev/null || echo '')
GO_LINT_URI=github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3
GO_SEC=$(shell command -v gosec 2> /dev/null || echo '')
GO_SEC_URI=github.com/securego/gosec/v2/cmd/gosec@v2.26.1
GO_VULNCHECK=$(shell command -v govulncheck 2> /dev/null || echo '')
GO_VULNCHECK_URI=golang.org/x/vuln/cmd/govulncheck@v1.6.0

default: test

env:
	@go version
test: test-full

test-fast: env
	go test -short ./...

test-full: env
	go test ./...

test-race: env
	go test -race -short ./...

test-coverage: env
	go test -cover -coverpkg ./... -coverprofile=coverage -covermode=set ./...
	@go tool cover -html=coverage -o coverage.html
	@go tool cover -func=coverage -o coverage.txt
	@tail -n 1 coverage.txt

vet: env
	go vet ./...

mockgen:
	$(if $(GO_MOCKGEN), ,go install $(GO_MOCKGEN_URI))
	@echo "##### Running mockgen"
	go generate ./...

lint:
	$(if $(GO_LINT), ,go install $(GO_LINT_URI))
	@echo "##### Running golangci-lint"
	golangci-lint run -D staticcheck -D unused --timeout=2m

sec:
	$(if $(GO_SEC), ,go install $(GO_SEC_URI))
	@echo "##### Running gosec"
	gosec -exclude-dir example ./...

vuln:
	$(if $(GO_VULNCHECK), ,go install $(GO_VULNCHECK_URI))
	@echo "##### Running govulncheck"
	govulncheck ./...

verify: test-full vet lint sec vuln
