# Build, deploy and debug the PocketBook Passbolt app. Everything that needs the
# PocketBook SDK runs in the pb-go-sdk Docker image (docker/Dockerfile).

IMAGE := pb-go-sdk
APP := build/passbolt.app
GDB_PORT := 10002

DOCKER := docker run --rm --platform linux/amd64 -v "$(CURDIR)":/src -w /src
# Named volumes keep Go's build and module caches between runs.
DOCKER_GO := $(DOCKER) -v pb-gocache:/root/.cache/go-build -v pb-gomod:/root/go/pkg/mod $(IMAGE)

.PHONY: image build debug deploy log gdb logos clean

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

# Logo images, generated from the official SVGs in assets/ with host tools
# (brew install librsvg imagemagick) and committed, so builds don't need them.
# E-ink has no red: the key is dark gray in the app and black on the launcher
# icon. Force a rebuild with make -B logos.
LOGOS := cmd/passbolt/images/logo-64.png cmd/passbolt/images/logo-148.png
ICONS := device/icons/passbolt.bmp device/icons/passbolt_f.bmp

logos: $(LOGOS) $(ICONS)

# In-app wordmark as an 8-bit grayscale PNG; the stem is its height in pixels.
cmd/passbolt/images/logo-%.png: assets/logo.svg
	@mkdir -p $(@D)
	sed 's/#[Ee]10600/#555555/g' $< | rsvg-convert --height $* --background-color white \
		| magick - -colorspace Gray -depth 8 -strip -define png:color-type=0 $@

# Launcher icons: uncompressed 8-bit BMP with a Windows 3.x header (BMP3).
# Don't reduce the colours: with 16 or fewer, ImageMagick writes a 4-bit BMP.
device/icons/passbolt.bmp: assets/icon.svg
	@mkdir -p $(@D)
	sed 's/#[Ee]10600/#000000/g' $< | rsvg-convert --width 96 --height 96 --background-color white \
		| magick - -gravity center -background white -extent 106x128 -colorspace Gray -type Palette -compress None BMP3:$@

# Focused (selected) icon: the same icon, inverted.
device/icons/passbolt_f.bmp: device/icons/passbolt.bmp
	magick $< -negate -colorspace Gray -type Palette -compress None BMP3:$@

clean:
	rm -rf build
