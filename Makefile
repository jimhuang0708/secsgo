# Makefile

GO      := go
GOPATH  := /var/go
export PATH := $(PATH):$(GOPATH)/bin

.PHONY: all webserver webhost clean

all: webserver webhost

webserver:
	@echo "==> Building webserver"
	cd src/webserver && \
	$(GO) get . && \
	$(GO) build .

webhost:
	@echo "==> Building webhost"
	cd src/webhost && \
	$(GO) get . && \
	$(GO) build .

clean:
	@echo "==> Cleaning binaries"
	rm -f src/webserver/webserver
	rm -f src/webhost/webhost
