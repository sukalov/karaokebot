FROM golang:1.23.4-alpine3.19 AS builder

WORKDIR /app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Build the application from the specific path
RUN CGO_ENABLED=0 GOOS=linux go build -o karaokebot ./cmd/karaokebot

# Final stage
FROM alpine:3.19

WORKDIR /root/

# Copy the pre-built binary
COPY --from=builder /app/karaokebot .

RUN useradd -m appuser
USER appuser

# Add HEALTHCHECK instruction
# Assumes the bot creates a pid file or has a way to check its status
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ps aux | grep -q 'karaokebot' || exit 1

# Run the bot
CMD ["./karaokebot"]