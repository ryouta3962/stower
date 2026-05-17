# --- Stage 1: Build the Go binary ---
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY go.* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o stower ./cmd

# --- Stage 2: Create the minimal execution environment ---
FROM alpine:3.19

# Stowerが裏側で使う必須コマンドをインストール
RUN apk add --no-cache docker-cli docker-cli-compose git

RUN git config --global --add safe.directory '*'

WORKDIR /app

COPY --from=builder /app/stower /app/stower
COPY --from=builder /app/public /app/public
RUN chmod +x /app/stower
RUN mkdir -p /app/workspace

LABEL org.opencontainers.image.source="https://github.com/ryouta3962/stower"

ENTRYPOINT ["/app/stower"]