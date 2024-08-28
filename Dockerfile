FROM golang:1.23-alpine AS builder

WORKDIR /app

ENV GOOS linux
ENV GOARCH amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o redis-benchmark .

FROM alpine:latest

COPY --from=builder /app/redis-benchmark /usr/local/bin/redis-benchmark

RUN chmod +x /usr/local/bin/redis-benchmark

ENTRYPOINT ["redis-benchmark"]
