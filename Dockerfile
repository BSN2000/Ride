# Build stage - use latest Go for compatibility
FROM golang:latest AS builder

WORKDIR /app

# Set GOTOOLCHAIN to auto to allow downloading newer toolchains
ENV GOTOOLCHAIN=auto

# Copy all files
COPY . .

# Download dependencies and tidy
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /ride-server ./cmd/server/main.go

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /ride-server .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./ride-server"]
