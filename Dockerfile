# Build Stage
FROM golang:1.21-alpine AS builder

# git is sometimes needed for go mod download
RUN apk add --no-cache git

WORKDIR /app

# Copy only mod file first
COPY go.mod ./

# Force generate go.sum and download dependencies
RUN go mod tidy
RUN go mod download

# Now copy the rest of the source
COPY . .

# Build the binary
# CGO_ENABLED=0 makes the build faster and lighter (pure Go)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Run Stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies for images and HTTPS
RUN apk add --no-cache \
    ca-certificates \
    font-dejavu \
    ttf-opensans

COPY --from=builder /app/main .
COPY --from=builder /app/assets ./assets

# Set Production Environment
ENV GIN_MODE=release
ENV PORT=8080

EXPOSE 8080

CMD ["./main"]