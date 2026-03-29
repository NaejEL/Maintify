# Use the official Go image as a parent image
FROM golang:1.24-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy core module for dependency resolution
COPY core/go.mod core/go.sum ./core/
COPY core/pkg/ ./core/pkg/

# Copy builder module
COPY builder/ ./builder/

# Set workdir to builder
WORKDIR /app/builder

# Download dependencies
# We run go mod tidy to ensure go.sum is generated and dependencies are correct
RUN go mod tidy

# Build the Go application
RUN go build -o /maintify-builder

# Expose port 8081 to the outside world
EXPOSE 8081

# Command to run the executable
CMD ["/maintify-builder"]
