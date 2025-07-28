# Build stage
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/cmd/reviewer

# Build to top-level /app location directly
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -a -installsuffix cgo -ldflags '-w -s' -o /app/reviewer-bot .

# Final image
FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY --from=builder /app/reviewer-bot .
COPY --from=builder /app/config ./config

RUN chmod +x /app/reviewer-bot
RUN ls -la /app  # 🔍 Helpful in CI

ENTRYPOINT ["/app/reviewer-bot"]
