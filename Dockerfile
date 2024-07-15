# Stage 1: Build the application
FROM golang:1.20.5 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o dist

# Debugging: List files in /app directory
RUN ls -al /app

# Stage 2: Create a lightweight production image
FROM alpine:latest

WORKDIR /app

# Copy the built executable from the builder stage
COPY --from=builder /app/dist /app/dist

# Debugging: List files in /app directory to verify the copy
RUN ls -al /app

# Ensure executable permissions (if needed)
RUN chmod +x /app/dist

# Create a non-root user
RUN adduser -D -g '' appuser

# Change to the non-root user
USER appuser

# Set environment variables
ENV PORT=8080

# Expose the port
EXPOSE 8080

# Command to run the application
CMD ["/app/dist"]
