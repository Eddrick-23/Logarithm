# Dev stage
FROM golang:1.25-alpine AS dev
WORKDIR /app
RUN go install github.com/air-verse/air@latest
COPY go.mod go.sum ./
RUN go mod download
COPY . .
CMD ["air", "-c", ".air.dashboard.toml"]

# global build base
FROM golang:1.25-alpine AS builder
WORKDIR /build
ENV CGO_ENABLED=0 GOOS=linux
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# TARGET: DASHBOARD
FROM alpine:latest AS dashboard
WORKDIR /root/
RUN --mount=type=bind,from=builder,source=/build,target=/build \
    go build -o /root/dashboard /build/cmd/dashboard
CMD ["./dashboard"]

# TARGET: INGESTER
FROM alpine:latest AS ingester
WORKDIR /root/
RUN --mount=type=bind,from=builder,source=/build,target=/build \
    go build -o /root/ingester /build/cmd/ingester
CMD ["./ingester"]

# TARGET: WORKER
FROM alpine:latest AS worker
WORKDIR /root/
RUN --mount=type=bind,from=builder,source=/build,target=/build \
    go build -o /root/worker /build/cmd/worker
CMD ["./worker"]
