# Build image for PocketBook B288 readers (InkPad 3) on firmware 6.x.
# The SDK's host tools are Linux x86_64 binaries, so the image is linux/amd64
# (runs under Rosetta on Apple silicon); `make image` builds it for that platform.
FROM debian:trixie-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
        ca-certificates curl git xz-utils file binutils-multiarch \
 && rm -rf /var/lib/apt/lists/*

# Official PocketBook SDK, branch "6.5" (the one for firmware 6.x), pinned.
# update_path.sh relocates the SDK's hardcoded /tmp/sdk-6.5 paths and
# switches the sysroot to the OpenSSL flavour of libcurl.
ARG SDK_COMMIT=dd8de463733a2bc0d53d1214b64734586fac427e
RUN git init -q /opt/pbsdk \
 && git -C /opt/pbsdk fetch -q --depth 1 https://github.com/pocketbook/SDK_6.3.0 "$SDK_COMMIT" \
 && git -C /opt/pbsdk checkout -q FETCH_HEAD \
 && rm -rf /opt/pbsdk/.git \
 && /opt/pbsdk/SDK-B288/bin/update_path.sh >/dev/null

ARG GO_VERSION=1.27.1
ARG GO_SHA256=63d339f0da5ab53635a56f2490a7984dfe12dfcff22ad749f63edaf590168445
RUN curl -fsSL -o /tmp/go.tgz "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
 && echo "${GO_SHA256}  /tmp/go.tgz" | sha256sum -c - \
 && tar -C /usr/local -xzf /tmp/go.tgz \
 && rm /tmp/go.tgz

COPY pb-cc /usr/local/bin/pb-cc

ENV SDK=/opt/pbsdk/SDK-B288
ENV SYSROOT=${SDK}/usr/arm-obreey-linux-gnueabi/sysroot
# The SDK's bin directory goes last so it never shadows the image's own tools.
ENV PATH=/usr/local/go/bin:${PATH}:${SDK}/usr/bin \
    GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 \
    CC=pb-cc \
    CGO_CFLAGS="-g -O2 -I${SYSROOT}/usr/local/include" \
    CGO_LDFLAGS="-g -O2 -L${SYSROOT}/usr/local/lib" \
    GOFLAGS=-buildvcs=false \
    GOTOOLCHAIN=local

WORKDIR /src
