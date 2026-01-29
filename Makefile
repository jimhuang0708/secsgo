# Makefile

GO      := go
GOPATH  := /var/go
export PATH := $(PATH):$(GOPATH)/bin

.PHONY: all webequipment webhost clean

all: webequipment webhost

webequipment:
	@echo "==> Building webequipment"
	cd src/webequipment && \
	$(GO) get . && \
	$(GO) build .

webhost:
	@echo "==> Building webhost"
	cd src/webhost && \
	$(GO) get . && \
	$(GO) build .

clean:
	@echo "==> Cleaning binaries"
	rm  src/webequipment/main
	rm  src/webhost/main
