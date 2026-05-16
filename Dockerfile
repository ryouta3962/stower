# プロジェクトルートに作成: Dockerfile

# --- Stage 1: Build the Go binary ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# 依存関係のダウンロード（今は何もないですが今後のために）
COPY go.mod ./
# go.sum はまだ無いかもしれないので、エラーを防ぐためにワイルドカードを使用
COPY go.* ./
RUN go mod download || true

# ソースコードをコピーしてビルド
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o stower ./cmd/main.go

# --- Stage 2: Create the minimal execution environment ---
FROM alpine:3.19

# Stowerが裏側で使う必須コマンドをインストール
RUN apk add --no-cache docker-cli docker-cli-compose git

WORKDIR /app

# Stage 1でビルドしたバイナリをコピー
COPY --from=builder /app/stower /app/stower

# 実行権限を付与（念のため）
RUN chmod +x /app/stower

# 設定ファイルと作業領域用のディレクトリを作成
RUN mkdir -p /app/workspace

# コンテナ起動時に実行されるコマンド
ENTRYPOINT ["/app/stower"]