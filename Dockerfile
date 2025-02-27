# Use the official Go image as the base image
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Copy the Firebase credentials file from the correct location
COPY cmd/api/keduapi-15ac7bd4be11.json /app/

# Build the application from cmd/api directory
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

# Start a new stage from scratch
FROM alpine:latest  

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Copy the public directory to serve static files
COPY --from=builder /app/public ./public

# Copy the Firebase credentials file to the final stage
COPY --from=builder /app/keduapi-15ac7bd4be11.json .

# Expose the port the app runs on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]