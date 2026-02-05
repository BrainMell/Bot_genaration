# Build Stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install system dependencies for cgo (required for some image ops)
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod ./
# If you had a go.sum, you'd copy it here. 
# Running go mod tidy will generate it if missing during build.

# Copy source code
COPY . .

# Download dependencies and Build
RUN go mod tidy
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Run Stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
# font-noto: standard fonts
# ca-certificates: for HTTPS requests (scraping)
RUN apk add --no-cache \
    ca-certificates \
    font-noto \
    ttf-freefont

COPY --from=builder /app/main .

# Expose port
EXPOSE 8080

# Run
CMD ["./main"]

