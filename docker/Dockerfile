FROM golang:1.23 AS builder

# Set the working directory
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the application
RUN go mod tidy && go build -o dist/gohead cmd/main.go

# Start with a minimal base image for the runtime
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/dist/gohead /app/gohead

# Expose the application port (adjust as needed)
EXPOSE 8080

# Command to run the application
CMD [ "/app/gohead" ]
