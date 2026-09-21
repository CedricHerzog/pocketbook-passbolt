# Build, deploy and debug the PocketBook Passbolt app. Everything that needs the
# PocketBook SDK runs in the pb-go-sdk Docker image (docker/Dockerfile).

IMAGE := pb-go-sdk
APP := build/passbolt.app
GDB_PORT := 10002

DOCKER := docker run --rm --platform linux/amd64 -v "$(CURDIR)":/src -w /src
# Named volumes keep Go's build and module caches between runs.
DOCKER_GO := $(DOCKER) -v pb-gocache:/root/.cache/go-build -v pb-gomod:/root/go/pkg/mod $(IMAGE)

.PHONY: image build debug deploy log gdb clean

image:
	docker build --platform linux/amd64 -t $(IMAGE) docker/

# Release build: optimised and stripped.
build:
	$(DOCKER_GO) go build -trimpath -ldflags="-s -w" -o $(APP) ./cmd/passbolt

# Debug build: no optimisation or inlining, symbols kept. Same output path, so
# the binary gdb reads is the one deployed to the reader.
debug:
	$(DOCKER_GO) go build -gcflags="all=-N -l" -o $(APP) ./cmd/passbolt

build/gdbserver:
	$(DOCKER) $(IMAGE) sh -c 'mkdir -p build && cp "$$SDK/usr/arm-obreey-linux-gnueabi/debug-root/usr/bin/gdbserver" build/'

deploy: build/gdbserver
	scripts/reader.sh deploy

log:
	scripts/reader.sh log

# Attach to passbolt-gdb.app running on the reader: make gdb IP=192.168.1.42
gdb:
	@test -n "$(IP)" || { echo "usage: make gdb IP=<reader IP>" >&2; exit 1; }
	$(DOCKER) -it $(IMAGE) gdb-multiarch -q -ix debug/gdbinit -ex "target remote $(IP):$(GDB_PORT)" $(APP)

clean:
	rm -rf build
