GO        := GO111MODULE=on go

.PHONY: list
list:
	@$(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | sort | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | grep -v go.mod | xargs

go.mod:
	$(GO) mod init

bin/revive: go.mod
	$(GO) build -o bin/revive github.com/mgechev/revive

bin/golangci-lint: go.mod
	$(GO) build -o bin/golangci-lint github.com/golangci/golangci-lint/cmd/golangci-lint
