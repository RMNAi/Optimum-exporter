# Stage 1: Build the Go application
FROM golang:alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY main.go .

# Build the binary (statically linked for the next stage)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o optimum-exporter .

# Stage 2: Create the minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS requests (if needed later)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/optimum-exporter .

# Expose the port the exporter runs on
EXPOSE 9100

# Command to run the executable
CMD ["./optimum-exporter"]