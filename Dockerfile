# Build stage with libpcap-dev installed
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy go mod and sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application's source code
COPY . .

# Build the application
RUN go build -a -o app .

# Final stage
FROM scratch AS final

COPY --from=builder /app/app /app

# Command to run
ENTRYPOINT ["/app"]
