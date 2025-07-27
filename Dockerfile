# ==== Build Stage ====
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build CLI and Server binaries
RUN go build -o cli ./cmd/reviewer
RUN go build -o server ./cmd/server

# ==== Runtime Stage ====
FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY --from=builder /app/cli ./cli
COPY --from=builder /app/server ./server
COPY --from=builder /app/config ./config

RUN chmod +x ./cli ./server


COPY <<EOF ./entrypoint.sh
#!/bin/sh
if [ "\$#" -eq 0 ]; then
  echo "🟢 Starting in SERVER mode"
  exec /app/server
else
  echo "🔵 Starting in CLI mode with args: \$@"
  exec /app/cli "\$@"
fi
EOF

RUN chmod +x ./entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]
