# --- Build Stage ---
FROM golang:1.21 AS builder

WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# --- Runtime Stage ---
FROM alpine:latest

# Install ca-certificates for HTTPS communication
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the built Go application from the builder stage
COPY --from=builder /app/main .

# Copy the frontend static files (assuming they are pre-built or built in a separate stage)
# You might need to copy these from a separate frontend build stage if not already in the Go repo
COPY frontend/dist ./frontend/dist 

# Expose the server port
EXPOSE 8080

# Set environment variables (for Docker, these should typically be passed at runtime)
# For demonstration, some defaults are provided.
ENV SERVER_PORT="8080"
# ENV DATABASE_URL="..."
# ENV KEYCLOAK_AUTH_SERVER_URL="..."
# ENV KEYCLOAK_REALM="..."
# ENV KEYCLOAK_ADMIN_CLIENT_ID="..."
# ENV KEYCLOAK_ADMIN_CLIENT_SECRET="..."
# ENV KEYCLOAK_FRONTEND_CLIENT_ID="..."
ENV FRONTEND_STATIC_PATH="./frontend/dist"


# Run the application
CMD ["./main"]
 