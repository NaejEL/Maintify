# Maintify Core Dockerfile

# Use a builder stage to compile the application
# Use a builder stage to compile the application
FROM golang:1.26-alpine AS builder
WORKDIR /app
# Copy module files first
COPY core/go.mod core/go.sum ./
# Copy all source files to match module structure
COPY core/pkg/ ./pkg/
COPY core/cmd/ ./cmd/
COPY core/migrations/ ./migrations/
COPY core/main.go ./

# Ensure dependencies are tidy (after all source is copied)
RUN go mod tidy
RUN go mod download

# Build the applications statically
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/maintify-core main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/rbac-cli cmd/rbac-cli/main.go

# Final, minimal image
FROM alpine:latest
WORKDIR /app
# Copy only the compiled binary from the builder stage
COPY --from=builder /app/maintify-core .
COPY --from=builder /app/rbac-cli .
COPY --from=builder /app/migrations ./migrations
# The plugins directory will be mounted as a volume in docker-compose, so it doesn't need to be copied.
EXPOSE 8080
CMD ["./maintify-core"]

