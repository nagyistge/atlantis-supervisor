PROJECT_ROOT := $(shell pwd)
ifeq ($(shell pwd | xargs dirname | xargs basename),lib)
	LIB_PATH := $(shell pwd | xargs dirname)
	VENDOR_PATH := $(shell pwd | xargs dirname | xargs dirname)/vendor
else
	LIB_PATH := $(PROJECT_ROOT)/lib
	VENDOR_PATH := $(PROJECT_ROOT)/vendor
endif
ATLANTIS_PATH := $(LIB_PATH)/atlantis

GOPATH := $(PROJECT_ROOT):$(VENDOR_PATH):$(ATLANTIS_PATH)
export GOPATH

all: test

clean:
	rm -rf bin pkg $(ATLANTIS_PATH)/src/atlantis/crypto/key.go
	rm -f example/supervisor example/client example/monitor

copy-key:
	@cp $(ATLANTIS_SECRET_DIR)/atlantis_key.go $(ATLANTIS_PATH)/src/atlantis/crypto/key.go

install-deps:
	@echo "Installing Dependencies..."
	@rm -rf $(LIB_PATH) $(VENDOR_PATH)
	@mkdir -p $(VENDOR_PATH) || exit 2
	@GOPATH=$(VENDOR_PATH) go get github.com/jigish/go-flags
	@GOPATH=$(VENDOR_PATH) go get github.com/BurntSushi/toml
	@GOPATH=$(VENDOR_PATH) go get launchpad.net/gocheck
	@git clone ssh://git@github.com/ooyala/atlantis $(ATLANTIS_PATH)
	@echo "Done."

test: clean copy-key
ifdef TEST_PACKAGE
	@echo "Testing $$TEST_PACKAGE..."
	@go test $$TEST_PACKAGE $$VERBOSE $$RACE
else
	@for p in `find ./src -type f -name "*.go" |sed 's-\./src/\(.*\)/.*-\1-' |sort -u`; do \
		[ "$$p" == 'atlantis/proxy' ] && continue; \
		echo "Testing $$p..."; \
		go test $$p || exit 1; \
	done
	@echo
	@echo "ok."
endif

.PHONY: example
example: copy-key
	@go build -o example/supervisor example/supervisor.go
	@go build -o example/client example/client.go
	@go build -o example/monitor example/monitor.go

fmt:
	@find src -name \*.go -exec gofmt -l -w {} \;
