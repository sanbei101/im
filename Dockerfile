FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum .

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    GOEXPERIMENT=jsonv2 CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o worker ./cmd/worker/ && \
    GOEXPERIMENT=jsonv2 CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o gateway ./cmd/gateway/ && \
    GOEXPERIMENT=jsonv2 CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api ./cmd/api/
    
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata bash

COPY --from=builder /app/gateway /app/gateway
COPY --from=builder /app/worker /app/worker
COPY --from=builder /app/api /app/api
COPY config.yaml /app/config.yaml

EXPOSE 8800

CMD ["/bin/bash", "-c", "/app/worker & /app/gateway & /app/api"]
