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

# Debugging: List files and check dist in builder stage
RUN ls -al /app
RUN ls -al /app/dist

# Stage 2: Create a lightweight production image
FROM alpine:latest

WORKDIR /app

# Copy the built executable from the builder stage
COPY --from=builder /app/dist /app/dist

# Ensure executable permissions
RUN chmod +x /app/dist

# Debugging: List files and check dist in final stage
RUN ls -al /app
RUN ls -al /app/dist

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
