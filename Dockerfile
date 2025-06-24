FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build both applications
RUN go build -o main main.go
RUN go build -o monitoring_server monitoring_server.go

FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the built applications
COPY --from=builder /app/main .
COPY --from=builder /app/monitoring_server .

# Create data directory
RUN mkdir -p data

# Expose monitoring server port
EXPOSE 8080

# Default command runs the monitoring server
CMD ["./monitoring_server"]
