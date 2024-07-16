# Use a minimal base image
FROM golang:1.20.5-alpine

# Set the working directory
WORKDIR /app

# Copy go modules
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o main

# Expose the port
EXPOSE 8080

# Command to run the application
CMD ["./main"]
