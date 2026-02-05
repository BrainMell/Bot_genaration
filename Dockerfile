# Build Stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy all source code first
COPY . .

# Now run tidy with the source code present
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Run Stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies for fonts
RUN apk add --no-cache \
    ca-certificates \
    font-dejavu \
    ttf-opensans

# Copy binary and assets from builder
COPY --from=builder /app/main .
COPY --from=builder /app/assets ./assets

# Set Production Environment
ENV GIN_MODE=release
ENV PORT=8080

EXPOSE 8080

CMD ["./main"]
