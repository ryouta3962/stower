# 📦 Stower

Stower is a lightweight, self-hosted CI/CD dashboard built with Go.
It automatically detects repository changes, builds your Docker images via `docker compose`, and pushes them to your registry.

## ✨ Features

*   **Lightweight & Fast:** Go 1.22 で書かれたバックエンドと、Vanilla JSによる軽量で高速なフロントエンド。
*   **Docker Compose Native:** 各プロジェクトの `compose.yml` をそのまま利用して `build` と `push` を実行。
*   **Simple Web UI:** プロジェクトの管理、手動ビルドの実行、リアルタイムなビルドログのストリーミング表示が可能。
*   **Polling Triggers:** 指定したインターバル（例: `1m`）でGitリポジトリを監視し、新しいコミットを検知して自動デプロイ。
*   **Secure Design:** GitやDockerレジストリのパスワード/トークンは画面から直接入力せず、サーバーの環境変数を参照する安全な設計。

## 🚀 Getting Started (本番環境でのデプロイ)

GitHub Container Registry (GHCR) の公式イメージを使用して、1つのコンテナとして簡単にデプロイできます。

### 1. `compose.yml` の作成

デプロイ先のサーバーに以下の `compose.yml` を作成します。

```yaml
services:
  stower:
    image: ghcr.io/ryouta3962/stower:latest
    container_name: stower-ci
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      # StowerからホストのDockerを操作するために必須
      - /var/run/docker.sock:/var/run/docker.sock
      # 設定ファイル(config.yml)やクローンしたソースを永続化
      - ./stower-data:/app/workspace
    environment:
      - TZ=Asia/Tokyo
      # 認証が必要な場合、以下にシークレットを環境変数として定義します
      # - MY_GIT_TOKEN=ghp_xxxxxxxxxxxx
      # - MY_DOCKER_PASS=xxxxxxxxxxxx
```

### 2. 起動

```bash
docker compose up -d
```

起動後、ブラウザで `http://<サーバーのIP>:8080` にアクセスすると、Stowerのダッシュボードが表示されます。

## 🛠️ Usage (プロジェクトの追加方法)

ダッシュボード右上の「New Project」からプロジェクトを登録します。

* **Repository URL:** GitリポジトリのURL (例: `https://github.com/ryouta3962/example.git`)
* **Branch:** 監視するブランチ名 (例: `main`)
* **Trigger Interval:** ポーリング間隔 (Goの `time.ParseDuration` 形式。例: `1m`, `30s`)
* **Registry (Optional):** プッシュ先のDockerレジストリ情報
* **Git Auth (Optional):** プライベートリポジトリをクローンするためのGit認証情報

> **💡 Note:**
> RegistryやGit Authの `Password Env Var` 項目には、パスワード本体ではなく、`compose.yml` の `environment` に定義した**環境変数の名前**（例: `MY_GIT_TOKEN`）を入力してください。

## 💻 Development (ローカル開発環境)

Stower自体の開発を行う場合、テスト用のローカルレジストリ（UI付き）を含む開発環境を一発で立ち上げることができます。

```bash
# リポジトリのクローン
git clone https://github.com/ryouta3962/stower.git
cd stower

# 開発用環境の起動（ローカルレジストリポート: 5000, レジストリUI: 8081）
docker compose up -d
```

* **Stower UI:** `http://localhost:8080`
* **Local Registry UI:** `http://localhost:8081`

## 📄 License

MIT License