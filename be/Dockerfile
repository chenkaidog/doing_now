# Stage 1: Build Stage
FROM golang:latest AS builder

WORKDIR /app

# Set environment variables for Go
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 creates a statically linked binary
# Use '.' to include all files in the package (including router.go where register might be defined)
RUN CGO_ENABLED=0 GOOS=linux go build -o output/main .

# Stage 2: Run Stage
FROM scratch

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/output/main .

# Copy configuration files
COPY --from=builder /app/conf ./conf

# Expose the port defined in main.go (8000)
EXPOSE 8000

# Command to run the executable
CMD ["./main"]
