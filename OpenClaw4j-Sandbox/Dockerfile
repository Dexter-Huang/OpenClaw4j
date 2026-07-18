FROM scratch AS sandlock-prebuilt

COPY third_party/sandlock/usr/local /usr/local
COPY third_party/sandlock/go /sandlock/go

FROM golang:1.26 AS go-build

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends pkg-config; \
    rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
COPY go.mod go.sum ./
COPY --from=sandlock-prebuilt /usr/local /usr/local
COPY --from=sandlock-prebuilt /sandlock/go /sandlock/go
ENV PKG_CONFIG_PATH=/usr/local/lib/pkgconfig \
    GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -o /usr/local/bin/openclaw4j-sandbox ./cmd/openclaw4j-sandbox

FROM python:3.12-slim-bookworm AS python-deps

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

FROM python:3.12-slim-bookworm AS runtime

COPY --from=go-build /usr/local/bin/openclaw4j-sandbox /usr/local/bin/openclaw4j-sandbox
COPY --from=sandlock-prebuilt /usr/local/lib/libsandlock_ffi.so /usr/local/lib/libsandlock_ffi.so
COPY --from=node-deps /usr/local/bin/node /usr/local/bin/node
COPY --from=python-deps /opt/openclaw4j-sandbox/deps/python /opt/openclaw4j-sandbox/deps/python
COPY --from=node-deps /opt/openclaw4j-sandbox/deps/node /opt/openclaw4j-sandbox/deps/node
RUN ldconfig

ENV OPENCLAW_SANDBOX_BIND=0.0.0.0:9010 \
    OPENCLAW_SANDBOX_DEPS_DIR=/opt/openclaw4j-sandbox/deps \
    OPENCLAW_SANDBOX_PYTHON_RUNTIME=/opt/openclaw4j-sandbox/deps/python/bin/python \
    OPENCLAW_SANDBOX_NODE_PATH=/opt/openclaw4j-sandbox/deps/node/node_modules
EXPOSE 9010

ENTRYPOINT ["/usr/local/bin/openclaw4j-sandbox"]
