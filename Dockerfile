# Etapa de build
FROM golang:1.24.4 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY server.go ./
RUN go build -o tcp-server server.go

# Etapa final
FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /app/tcp-server .

EXPOSE 12345

CMD ["./tcp-server"]
