GOVERSION=$(shell go version)
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
VERSION=$(patsubst "%",%,$(lastword $(shell grep 'const Version' version.go)))
ARTIFACTS_DIR=./artifacts/$(VERSION)
RELEASE_DIR=./release/$(VERSION)
SRC_FILES = $(wildcard *.go ./cmd/*.go ntpdshm/*.go)
GITHUB_USERNAME=shogo82148

.PHONY: help
help: ## Show this text.
	# https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## run tests
	go test -v -race -covermode=atomic -coverprofile=coverage.out ./...

all: build-windows-386 build-windows-amd64 build-linux-386 build-linux-amd64 build-darwin-386 build-darwin-amd64 ## buidls all binary

$(ARTIFACTS_DIR)/webntp_$(GOOS)_$(GOARCH):
	@mkdir -p $@

$(ARTIFACTS_DIR)/webntp_$(GOOS)_$(GOARCH)/webntp$(SUFFIX): $(ARTIFACTS_DIR)/webntp_$(GOOS)_$(GOARCH) $(SRC_FILES)
	@echo " * Building binary for $(GOOS)/$(GOARCH)..."
	@./run-in-docker.sh go build -o $@ ./cmd/webntp/main.go

build: $(ARTIFACTS_DIR)/webntp_$(GOOS)_$(GOARCH)/webntp$(SUFFIX)

build-windows-amd64:
	@$(MAKE) build GOOS=windows GOARCH=amd64 SUFFIX=.exe

build-windows-386:
	@$(MAKE) build GOOS=windows GOARCH=386 SUFFIX=.exe

build-linux-amd64:
	@$(MAKE) build GOOS=linux GOARCH=amd64

build-linux-386:
	@$(MAKE) build GOOS=linux GOARCH=386

build-darwin-amd64:
	@$(MAKE) build GOOS=darwin GOARCH=amd64

build-darwin-386:
	@$(MAKE) build GOOS=darwin GOARCH=386

##### release settings

.PHONY: release-windows-amd64 release-windows-386 release-linux-amd64 release-linux-386 release-darwin-amd64 release-darwin-386
.PHONY: release-targz release-zip release-files release-upload

$(RELEASE_DIR)/webntp_$(GOOS)_$(GOARCH):
	@mkdir -p $@

release-windows-amd64:
	@$(MAKE) release-zip GOOS=windows GOARCH=amd64 SUFFIX=.exe

release-windows-386:
	@$(MAKE) release-zip GOOS=windows GOARCH=386 SUFFIX=.exe

release-linux-amd64:
	@$(MAKE) release-targz GOOS=linux GOARCH=amd64

release-linux-386:
	@$(MAKE) release-targz GOOS=linux GOARCH=386

release-darwin-amd64:
	@$(MAKE) release-zip GOOS=darwin GOARCH=amd64

release-darwin-386:
	@$(MAKE) release-zip GOOS=darwin GOARCH=386

release-targz: build $(RELEASE_DIR)/webntp_$(GOOS)_$(GOARCH)
	@echo " * Creating tar.gz for $(GOOS)/$(GOARCH)"
	tar -czf $(RELEASE_DIR)/webntp_$(GOOS)_$(GOARCH).tar.gz -C $(ARTIFACTS_DIR) webntp_$(GOOS)_$(GOARCH)

release-zip: build $(RELEASE_DIR)/webntp_$(GOOS)_$(GOARCH)
	@echo " * Creating zip for $(GOOS)/$(GOARCH)"
	cd $(ARTIFACTS_DIR) && zip -9 ../../$(RELEASE_DIR)/webntp_$(GOOS)_$(GOARCH).zip webntp_$(GOOS)_$(GOARCH)/*

release-files: release-windows-386 release-windows-amd64 release-linux-386 release-linux-amd64 release-darwin-386 release-darwin-amd64

release-upload: release-files
	ghr -u $(GITHUB_USERNAME) --draft --replace v$(VERSION) $(RELEASE_DIR)

clean:
	-rm -rf artifacts
	-rm -rf release
	-rm -rf .mod
