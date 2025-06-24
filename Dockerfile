FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git and other dependencies
RUN apk add --no-cache git

# Copy source code first
COPY . .

# Initialize go module if it doesn't exist
RUN if [ ! -f go.mod ]; then go mod init datasoup; fi

# Download dependencies
RUN go mod tidy
RUN go mod download

# Build both applications
RUN go build -o main main.go
RUN go build -o monitoring_server monitoring_server.go

FROM alpine:latest

# Install ca-certificates
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
