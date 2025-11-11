FROM golang:1.24.6-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /build/app \
    ./cmd/main.go

# ---------------------------
# Runtime stage
# ---------------------------
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata netcat-openbsd
WORKDIR /app
COPY --from=builder /build/app /app/app
RUN mkdir -p /app/logs

# Create wait-for script
RUN echo '#!/bin/sh' > /app/wait-for.sh && \
    echo 'set -e' >> /app/wait-for.sh && \
    echo 'host="$1"' >> /app/wait-for.sh && \
    echo 'port="$2"' >> /app/wait-for.sh && \
    echo 'shift 2' >> /app/wait-for.sh && \
    echo 'echo "Waiting for $host:$port..."' >> /app/wait-for.sh && \
    echo 'until nc -z "$host" "$port"; do' >> /app/wait-for.sh && \
    echo '  echo "  $host:$port is unavailable - sleeping"' >> /app/wait-for.sh && \
    echo '  sleep 1' >> /app/wait-for.sh && \
    echo 'done' >> /app/wait-for.sh && \
    echo 'echo "  $host:$port is up - executing command"' >> /app/wait-for.sh && \
    echo 'exec "$@"' >> /app/wait-for.sh && \
    chmod +x /app/wait-for.sh

CMD ["/app/app"]
