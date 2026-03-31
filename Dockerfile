FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vortex .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/vortex .
COPY --from=builder /app/vortex.yaml .

EXPOSE 8000

CMD ["./vortex"]
