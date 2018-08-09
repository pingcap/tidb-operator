PD_PKG := github.com/pingcap/pd

TEST_PKGS := $(shell find . -iname "*_test.go" -exec dirname {} \; | \
                     uniq | sed -e "s/^\./github.com\/pingcap\/pd/")
BASIC_TEST_PKGS := $(filter-out github.com/pingcap/pd/pkg/integration_test,$(TEST_PKGS))

GOFILTER := grep -vE 'vendor|testutil'
GOCHECKER := $(GOFILTER) | awk '{ print } END { if (NR > 0) { exit 1 } }'

LDFLAGS += -X "$(PD_PKG)/server.PDReleaseVersion=$(shell git describe --tags --dirty)"
LDFLAGS += -X "$(PD_PKG)/server.PDBuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "$(PD_PKG)/server.PDGitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "$(PD_PKG)/server.PDGitBranch=$(shell git rev-parse --abbrev-ref HEAD)"

# Ignore following files's coverage.
#
# See more: https://godoc.org/path/filepath#Match
COVERIGNORE := "cmd/*/*,pdctl/*,pdctl/*/*,server/api/bindata_assetfs.go"

default: build

all: dev

dev: build simulator check test

ci: build simulator check basic_test

build:
ifeq ("$(WITH_RACE)", "1")
	CGO_ENABLED=1 go build -race -ldflags '$(LDFLAGS)' -o bin/pd-server cmd/pd-server/main.go
else
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o bin/pd-server cmd/pd-server/main.go
endif
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o bin/pd-ctl cmd/pd-ctl/main.go
	CGO_ENABLED=0 go build -o bin/pd-tso-bench cmd/pd-tso-bench/main.go
	CGO_ENABLED=0 go build -o bin/pd-recover cmd/pd-recover/main.go

test:
	# testing..
	CGO_ENABLED=1 go test -race -cover $(TEST_PKGS)

basic_test:
	go test $(BASIC_TEST_PKGS)

check:
	go get github.com/golang/lint/golint

	@echo "vet"
	@ go tool vet . 2>&1 | $(GOCHECKER)
	@ go tool vet --shadow . 2>&1 | $(GOCHECKER)
	@echo "golint"
	@ golint ./... 2>&1 | $(GOCHECKER)
	@echo "gofmt"
	@ gofmt -s -l . 2>&1 | $(GOCHECKER)

travis_coverage:
ifeq ("$(TRAVIS_COVERAGE)", "1")
	GOPATH=$(VENDOR) $(HOME)/gopath/bin/goveralls -service=travis-ci -ignore $(COVERIGNORE)
else
	@echo "coverage only runs in travis."
endif

update:
	which dep 2>/dev/null || go get -u github.com/golang/dep/cmd/dep
ifdef PKG
	dep ensure -add ${PKG}
else
	dep ensure -update
endif
	@echo "removing test files"
	dep prune
	bash ./hack/clean_vendor.sh

simulator:
	CGO_ENABLED=0 go build -o bin/simulator cmd/simulator/main.go

.PHONY: update clean
