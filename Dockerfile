# Build stage
FROM golang:1.23-alpine AS builder

# Install required build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled
ENV CGO_ENABLED=1
RUN go build -ldflags="-s -w" -o main .

# Final stage - use minimal alpine image
FROM alpine:3.19

# Install CA certificates for HTTPS and any runtime dependencies
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy only the compiled binary from the build stage
COPY --from=builder /build/main .

EXPOSE 8080

# Run the Go binary
CMD ["./main"]