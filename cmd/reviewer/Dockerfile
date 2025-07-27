# Build stage
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go.mod and go.sum from root (since build context is repo root)
COPY go.mod go.sum ./

RUN go mod download

# Copy full source code from root
COPY . .

WORKDIR /app/cmd/reviewer

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s' -o reviewer-bot .

# Final image
FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY --from=builder /app/cmd/reviewer/reviewer-bot .

COPY --from=builder /app/config ./config

RUN chmod +x reviewer-bot

ENTRYPOINT ["./reviewer-bot"]
