# Build and deploy the PocketBook Passbolt app. Everything that needs the
# PocketBook SDK runs in the pb-go-sdk Docker image (docker/Dockerfile).

IMAGE := pb-go-sdk
APP := build/passbolt.app

DOCKER := docker run --rm --platform linux/amd64 -v "$(CURDIR)":/src -w /src
# Named volumes keep Go's build and module caches between runs.
DOCKER_GO := $(DOCKER) -v pb-gocache:/root/.cache/go-build -v pb-gomod:/root/go/pkg/mod $(IMAGE)

.PHONY: image build deploy preview logos blog memes clean

image:
	docker build --platform linux/amd64 -t $(IMAGE) docker/

# Release build: optimised and stripped.
build:
	$(DOCKER_GO) go build -trimpath -ldflags="-s -w" -o $(APP) ./cmd/passbolt

deploy:
	scripts/reader.sh deploy

# The app's screens on one overview image in build/preview/, rendered on the
# host (needs Go, no Docker or reader) with made-up data. The real drawing
# code runs against a stand-in inkview (sim/inkview), swapped in through a
# copy of go.mod, with copies of the reader's fonts. Other screen sizes:
# PB_SIZE=1072x1448 make preview.
preview:
	@mkdir -p build
	cp go.mod build/sim.mod
	cp go.sum build/sim.sum
	go mod edit -modfile=build/sim.mod -replace github.com/dennwc/inkview=./sim/inkview
	CGO_ENABLED=0 go run -modfile=build/sim.mod -mod=mod -tags sim ./cmd/passbolt -eink

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

# Blog post (docs/blog) as a PDF, and each of its diagrams as an SVG for the
# blog's CMS. Needs typst on the Mac (brew install typst). To render a single
# diagram: typst compile --input d=<name> docs/blog/render.typ <name>.svg
DIAGRAMS := $(basename $(notdir $(wildcard docs/blog/diagrams/*.typ)))

blog:
	@mkdir -p build/blog
	typst compile docs/blog/post.typ build/blog/post.pdf
	for d in $(DIAGRAMS); do \
		typst compile --input d=$$d docs/blog/render.typ build/blog/$$d.svg || exit 1; \
	done

# Memes for the blog post (docs/blog/memes): each one as a normal and an e-ink
# version, and both side by side, as PNGs in build/blog/memes/. The e-ink
# images are made from the meme templates with ImageMagick (16 grays, dithered,
# in the panel's ink and paper colours) and committed, like the logos. To render
# a single meme: typst compile --input m=<name> [--input v=normal|eink]
# docs/blog/render-meme.typ <name>.png
MEMES := $(basename $(notdir $(wildcard docs/blog/memes/templates/*)))
EINK = magick $< -colorspace Gray -level 4%,96% -dither FloydSteinberg -colors 16 \
	+level-colors '\#262626','\#e8e5dc' -strip $@

memes: $(MEMES:%=docs/blog/memes/eink/%.png)
	@mkdir -p build/blog/memes
	for m in $(MEMES); do \
		typst compile --ppi 144 --input m=$$m docs/blog/render-meme.typ build/blog/memes/$$m.png || exit 1; \
		for v in normal eink; do \
			typst compile --ppi 144 --input m=$$m --input v=$$v docs/blog/render-meme.typ build/blog/memes/$$m-$$v.png || exit 1; \
		done; \
	done

docs/blog/memes/eink/%.png: docs/blog/memes/templates/%.jpg
	@mkdir -p $(@D)
	$(EINK)

docs/blog/memes/eink/%.png: docs/blog/memes/templates/%.png
	@mkdir -p $(@D)
	$(EINK)

clean:
	rm -rf build
