# syntax=docker/dockerfile:1

FROM golang:1.25.6-alpine AS builder

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    git \
    tzdata

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 \
    GOOS=linux \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/sgscms-api \
    ./cmd/api

RUN CGO_ENABLED=0 \
    GOOS=linux \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/sgscms-seed \
    ./cmd/seed

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup \
    && mkdir -p /app/uploads \
    && chown -R appuser:appgroup /app

COPY --from=builder \
    /out/sgscms-api \
    /app/sgscms-api

COPY --from=builder \
    /out/sgscms-seed \
    /app/sgscms-seed

USER appuser

EXPOSE 8080

HEALTHCHECK \
    --interval=30s \
    --timeout=5s \
    --start-period=10s \
    --retries=3 \
    CMD wget --quiet --spider http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/app/sgscms-api"]