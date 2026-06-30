FROM node:22-alpine AS frontend

WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# ---

FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /app/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o abacus ./cmd/abacus

# ---

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/abacus .

# Run as a non-root user. Pre-create the data dir owned by that user so a
# bind/named volume mounted there inherits writable ownership on first use.
RUN adduser -D -H -u 10001 abacus \
    && mkdir -p /app/data \
    && chown -R abacus:abacus /app
USER abacus

EXPOSE 8080

# Liveness probe against the unauthenticated health endpoint (busybox wget).
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/v1/health || exit 1

CMD ["./abacus"]
