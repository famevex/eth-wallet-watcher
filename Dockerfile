# Stage 1 — build
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bot ./cmd/bot/main.go

# Stage 2 — final image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bot .

CMD ["./bot"]