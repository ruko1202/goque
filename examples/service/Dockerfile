# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
WORKDIR /app/examples/service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/service ./cmd/service

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary
COPY --from=builder /app/bin/service .

# Copy web assets
COPY --from=builder /app/examples/service/web ./web

# Copy config
COPY --from=builder /app/examples/service/config.yaml .

EXPOSE 8080

CMD ["./service"]
