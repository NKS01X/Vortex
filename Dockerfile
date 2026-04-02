# Stage 1: Builder
FROM golang:alpine AS builder
WORKDIR /build

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vortex .

# Stage 2: Development (Hot-Reloading with Air)
FROM golang:alpine AS dev
WORKDIR /app

# Install Air
RUN go install github.com/air-verse/air@latest

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

EXPOSE 8000
CMD ["air"]

# Stage 3: Production (Standard Start)
FROM alpine:latest AS prod
WORKDIR /root/

# Copy the built binary and config
COPY --from=builder /build/vortex .
COPY --from=builder /build/vortex.yaml .

EXPOSE 8000
CMD ["./vortex"]
