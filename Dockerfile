FROM golang:1.22 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o monke-proxy

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/monke-proxy .

# Copy .env file - note that in production you might want to use environment variables instead
COPY .env .

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
CMD ["./monke-proxy"]