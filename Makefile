GO		= go
PKGS	= $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./...))

DATE		= $(shell date +%FT%T%z)
LDFLAGS	= -ldflags "-X main.sha1ver=$(VERSION) -X main.buildTime=${DATE}"

V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\e[94m-->\e[39m")

SRC		= main.go
BIN		= bin
BINARY	= dnsquerylog

VERSION ?= $(shell git describe --tags --always --dirty --match=v*)
BUILDFLAGS=${LDFLAGS} -buildmode=exe

build: vet tidy
	go build ${BUILDFLAGS} -o ${BIN}/${BINARY} ${SRC}

#generate-golden:
#	go test integration/cli_test.go -update

install: vet tidy # test
	go install ${BUILDFLAGS}

vet:
	go vet ${SRC}

tidy:
	go mod tidy

run:
	go run ${SRC}

compile: build

msi: windows-64
	go-msi make --msi bin/wlctl.msi --version ${VERSION}

all: build linux-64 windows-64 macos-64 install

#freebsd-64:
#	GOOS=freebsd GOARCH=amd64 go build ${BUILDFLAGS} -o bin/${BINARY}-freebsd-amd64 ${SRC}

macos-64:
	GOOS=darwin GOARCH=amd64 go build ${BUILDFLAGS} -o bin/${BINARY}-darwin-amd64 ${SRC}

linux-64:
	GOOS=linux GOARCH=amd64 go build ${BUILDFLAGS} -o bin/${BINARY}-linux-amd64 ${SRC}

windows-64:
	GOOS=windows GOARCH=amd64 go build ${BUILDFLAGS} -o bin/${BINARY}.exe ${SRC}
#	GOOS=windows GOARCH=amd64 go build ${BUILDFLAGS} -o bin/${BINARY}-windows-amd64.exe ${SRC}


TIMEOUT  = 20
PKGS     = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./...))
TESTPKGS = $(shell env GO111MODULE=on $(GO) list -f \
			'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
			$(PKGS))

TEST_TARGETS := test-default test-bench test-short test-verbose test-race
test-bench: ARGS=-run=__absolutelynothing__ -bench=.
test-short:	ARGS=-short
test-verbose: ARGS=-v
test-race: ARGS=-race
$(TEST_TARGETS): test

check tests: fmt test
#	go test -timeout $(TIMEOUT)s $(ARGS) $(TESTPKGS)

test: build windows-64 linux-64
	-mkdir test
	echo "${TESTPKGS}" | xargs -t -n 1 go test -o 'test/testworkaround' -timeout $(TIMEOUT)s $(ARGS)
	-rm test/testworkaround

.PHONY: fmt
fmt: ; $(info $(M) running gofmt) @ ## Run gofmt on all source files
	$Q $(GO) fmt $(PKGS)

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning)	@ ## Cleanup everything
	@rm -rf $(BIN)
	@rm -rf test/tests.* test/coverage.*

$(BIN):
	-mkdir -p ${BIN}
