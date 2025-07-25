# Build stage
FROM golang:1.24.4-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-w -s' -o ai-code-review-bot .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY --from=builder /app/ai-code-review-bot .

COPY --from=builder /app/config/ ./config/

RUN chmod +x ./ai-code-review-bot

# Run the application
ENTRYPOINT ["/app/ai-code-review-bot"]