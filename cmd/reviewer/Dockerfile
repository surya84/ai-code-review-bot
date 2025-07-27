# Build stage
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy source
COPY . .

# Set working directory to where main.go is located
WORKDIR /app/cmd/reviewer

# Cross-compile for Linux
RUN GOOS=linux GOARCH=amd64 go build -o /app/reviewer-bot .

# Final image
FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY --from=builder /app/reviewer-bot .
COPY --from=builder /app/config ./config

RUN chmod +x reviewer-bot

ENTRYPOINT ["./reviewer-bot"]