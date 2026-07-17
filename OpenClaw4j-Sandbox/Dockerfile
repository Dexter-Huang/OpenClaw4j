FROM rust:1.96 AS sandlock-ffi-build

WORKDIR /sandlock
RUN git clone --depth 1 https://github.com/multikernel/sandlock.git .
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/usr/local/cargo/git \
    --mount=type=cache,target=/sandlock/target \
    set -eux; \
    cargo build --release -p sandlock-ffi; \
    mkdir -p /sandlock-install/lib /sandlock-install/include /sandlock-install/lib/pkgconfig; \
    cp /sandlock/target/release/libsandlock_ffi.so /sandlock-install/lib/libsandlock_ffi.so; \
    cp /sandlock/crates/sandlock-ffi/include/sandlock.h /sandlock-install/include/sandlock.h; \
    sed "s|@PREFIX@|/usr/local|g; s|@LIBDIR@|/usr/local/lib|g; s|@INCLUDEDIR@|/usr/local/include|g" \
        /sandlock/go/sandlock.pc.in > /sandlock-install/lib/pkgconfig/sandlock.pc

FROM golang:1.24 AS go-build

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends pkg-config; \
    rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
COPY go.mod go.sum ./
COPY --from=sandlock-ffi-build /sandlock-install /usr/local
COPY --from=sandlock-ffi-build /sandlock/go /sandlock/go
ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig \
    GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -o /usr/local/bin/openclaw4j-sandbox ./cmd/openclaw4j-sandbox

FROM python:3.12-slim AS python-deps

COPY deps/python/requirements.txt /deps/python/requirements.txt
RUN --mount=type=cache,target=/root/.cache/pip \
    set -eux; \
    python3 -m venv /opt/openclaw4j-sandbox/deps/python; \
    /opt/openclaw4j-sandbox/deps/python/bin/python -m pip install --upgrade pip setuptools wheel; \
    /opt/openclaw4j-sandbox/deps/python/bin/pip install -r /deps/python/requirements.txt

FROM node:22-bookworm-slim AS node-deps

COPY deps/node/package.json deps/node/pnpm-lock.yaml /deps/node/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    set -eux; \
    corepack enable; \
    corepack prepare pnpm@8.6.12 --activate; \
    pnpm install --prod --frozen-lockfile --dir /deps/node; \
    mkdir -p /opt/openclaw4j-sandbox/deps/node; \
    cp -a /deps/node/node_modules /opt/openclaw4j-sandbox/deps/node/node_modules; \
    cp /deps/node/package.json /deps/node/pnpm-lock.yaml /opt/openclaw4j-sandbox/deps/node/

FROM debian:bookworm-slim AS runtime

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates nodejs python3 python3-venv; \
    rm -rf /var/lib/apt/lists/*

COPY --from=go-build /usr/local/bin/openclaw4j-sandbox /usr/local/bin/openclaw4j-sandbox
COPY --from=sandlock-ffi-build /sandlock-install/lib/libsandlock_ffi.so /usr/local/lib/libsandlock_ffi.so
COPY --from=python-deps /opt/openclaw4j-sandbox/deps/python /opt/openclaw4j-sandbox/deps/python
COPY --from=node-deps /opt/openclaw4j-sandbox/deps/node /opt/openclaw4j-sandbox/deps/node
RUN ldconfig

ENV OPENCLAW_SANDBOX_BIND=0.0.0.0:9010 \
    OPENCLAW_SANDBOX_DEPS_DIR=/opt/openclaw4j-sandbox/deps \
    OPENCLAW_SANDBOX_PYTHON_RUNTIME=/opt/openclaw4j-sandbox/deps/python/bin/python \
    OPENCLAW_SANDBOX_NODE_PATH=/opt/openclaw4j-sandbox/deps/node/node_modules
EXPOSE 9010

ENTRYPOINT ["/usr/local/bin/openclaw4j-sandbox"]
