FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -o ./bin/dashboard ./cmd/dashboard
RUN go build -o ./bin/ingester ./cmd/ingester
RUN go build -o ./bin/worker ./cmd/worker

FROM alpine:latest 
WORKDIR /root/
COPY --from=builder /build/bin .

# no start commands, handled by docker compose
