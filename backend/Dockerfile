FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY . .
RUN go mod download

RUN ls -la
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -o ./bin/dashboard cmd/dashboard/main.go
RUN go build -o ./bin/ingester cmd/ingester/main.go
RUN go build -o ./bin/worker cmd/worker/main.go

FROM alpine:latest 
WORKDIR /root/
COPY --from=builder /build/bin .

# no start commands, handled by docker compose
