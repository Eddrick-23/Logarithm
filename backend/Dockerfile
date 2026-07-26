# Dev stage
FROM golang:1.25-alpine AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["air", "-c", ".air.dashboard.toml"]

# global build base
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
WORKDIR /build
ENV CGO_ENABLED=0 GOOS=linux

ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/go/pkg/mod

ARG TARGETARCH

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# isolated build stages
FROM builder AS build-dashboard
RUN --mount=type=bind,source=.,target=/build \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOARCH=$TARGETARCH go build -o /out/dashboard ./cmd/dashboard

FROM builder AS build-ingester
RUN --mount=type=bind,source=.,target=/build \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOARCH=$TARGETARCH go build -o /out/ingester ./cmd/ingester

FROM builder AS build-worker
RUN --mount=type=bind,source=.,target=/build \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOARCH=$TARGETARCH go build -o /out/worker ./cmd/worker

# TARGET: DASHBOARD
FROM alpine:latest AS dashboard
WORKDIR /root/
COPY --from=build-dashboard /out/dashboard .
CMD ["./dashboard"]

# TARGET: INGESTER
FROM alpine:latest AS ingester
WORKDIR /root/
COPY --from=build-ingester /out/ingester .
CMD ["./ingester"]

# TARGET: WORKER
FROM alpine:latest AS worker
WORKDIR /root/
COPY --from=build-worker /out/worker .
CMD ["./worker"]
