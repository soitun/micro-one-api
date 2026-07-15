# Multi-stage build for security
# Usage:
#   docker build -f app/log/Dockerfile ../.. \
#                --build-arg SERVICE_NAME=log-service \
#                --build-arg SERVICE_PATH=./app/log/cmd/log
ARG SERVICE_NAME=log-service
ARG SERVICE_PATH=./app/log/cmd/log

# ---------------------------------------------------------------------------
# Stage 2: go-deps
# Caches Go modules inside this Dockerfile so CI/buildx jobs are self-contained.
# A separate Dockerfile.deps may still be used by deployment scripts for
# coarse-grained prebuild caching, but this image must not require a local tag.
# ---------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS go-deps

RUN apk add --no-cache git ca-certificates
ENV GOPROXY=https://proxy.golang.org|https://goproxy.cn|direct

WORKDIR /deps

COPY go.mod go.sum ./

RUN go mod download && go mod verify

# ---------------------------------------------------------------------------
# Stage 3: go-builder
# Builds the Go service binary. This stage runs on the TARGET platform
# (natively, or emulated via QEMU on an amd64 GitHub runner), NOT on the
# build platform. That way the CGO toolchain (mattn/go-sqlite3) always
# matches the target arch and no fragile cross-sysroot is needed.
# docker/setup-qemu-action (configured in the CI workflow) supplies the
# emulation, so linux/arm64 is handled transparently.
# ---------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

# build-base pulls gcc + musl-dev for CGO (mattn/go-sqlite3); sqlite-dev
# provides the headers/libs libsqlite3 links against, and sqlite-libs is
# the runtime dep shipped in the final alpine image.
RUN apk add --no-cache \
      git ca-certificates make protobuf protobuf-dev \
      build-base sqlite-libs sqlite-dev
ENV GOPROXY=https://proxy.golang.org|https://goproxy.cn|direct

WORKDIR /app

# Copy Go module cache from the self-contained deps stage. CI builds each
# service in a fresh buildx job, so relying on a pre-existing local image would
# make BuildKit try to pull micro-one-api/go-deps from Docker Hub.
COPY --from=go-deps /go/pkg/mod /go/pkg/mod

COPY . .

RUN make proto-tools && make proto


# CGO is required by github.com/mattn/go-sqlite3 so the lite deployment
# (DATABASE_DRIVER=sqlite3) works. Because this stage runs on the target
# platform, Go's host detection yields the correct GOOS/GOARCH and the
# platform-native gcc is used — no manual CC/CXX selection is needed.
ARG SERVICE_PATH
RUN CGO_ENABLED=1 go build \
      -ldflags='-s -w -extldflags "-static"' \
      -o /app/bin/service ${SERVICE_PATH}

# ---------------------------------------------------------------------------
# Stage 4: runtime
# Minimal alpine image with sqlite-libs (CGO-linked binaries need musl
# libc + libsqlite3 at runtime). Runs as the unprivileged 65534 user.
# ---------------------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/bin/service /service
COPY --from=builder /app/app/log/configs /configs

# Default config path; overridable via -e CONF_PATH=... at runtime.
ENV CONF_PATH=/configs/config.yaml

USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/service"]
