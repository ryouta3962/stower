# 🚢 Stower CI

[日本語版は下にあります (Japanese version is available below)](#-日本語-japanese)

Stower is a lightweight, self-hostable, Docker Compose-native CI/CD tool.
It monitors target repositories, detects changes, and automatically executes `docker compose build` and `docker compose push`. It features an intuitive Web UI that allows real-time viewing of build logs and manual execution.

## ✨ Features

- **Docker Compose Native**: Uses the `compose.yml` of the monitored repository as the build instructions as-is.
- **Simple Web UI**: Project management, status checking, and real-time log monitoring are all completed in the browser.
- **Flexible Triggers**: Supports Git repository polling (periodic monitoring) and manual triggers.
- **Secure Authentication Management**: Securely handles authentication credentials for private Git repositories and private Docker container registries via environment variables.

---

## 🚀 Installation & Getting Started

Stower itself is provided as a Docker container. Please place the following `compose.yml` in your production (or local) environment and start it.

```yaml
services:
  stower:
    image: ghcr.io/<YOUR_GITHUB_USERNAME>/stower:latest
    container_name: stower-ci
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./stower-data:/app/workspace
    environment:
      - TZ=Asia/Tokyo
      # Example: Set tokens for private repositories or passwords for registries
      # - MY_GIT_TOKEN=ghp_xxxxxxxxx
      # - DOCKER_REGISTRY_PASS=xxxxxxxxx
```

```bash
docker compose pull
docker compose up -d
```

After starting, access `http://<SERVER_IP>:8080` in your browser to view the dashboard.

## 📖 Usage

Here are the steps to automatically build and push your projects using Stower.

### 1. Prepare the Monitored Repository

Stower performs builds and pushes using the `compose.yml` located in the root directory of the target repository. Therefore, you need to define both `build` and `image` in the `compose.yml` within the repository.

Example: `compose.yml` of the monitored repository

```yaml
services:
  web:
    # Target for `docker compose build` executed by Stower
    build: .
    # Destination for `docker compose push` executed by Stower after building (Registry URL/Image Name:Tag)
    image: localhost:5000/test-web:latest
    ports:
      - "8080:80"
```
*Note: Ensure you include the push destination registry address in `image:` (it can be omitted for Docker Hub).*

### 2. Add a Project (Web UI)

Click "New Project" in the upper right corner of the Stower dashboard to register the project you want to monitor.

**Basic Settings**
- **Repository URL**: Git repository URL to monitor (e.g., `https://github.com/user/repo.git`)
- **Branch**: Branch name to monitor (e.g., `main`)

**Trigger Settings**
- **Type**: Select `polling` (periodic monitoring).
- **Interval**: Specify the monitoring interval. (e.g., `1m` = 1 minute, `30s` = 30 seconds)

**🔐 Authentication Settings (Optional)**

**[Registry (Docker Registry)]**
Configure this if the push destination is a private registry.
- **Server**: Registry server address (e.g., `ghcr.io` or `localhost:5000`)
- **Username**: Registry username
- **Password Env Var**: The name of the environment variable in the Stower container where the password or token is stored. (e.g., If you set `DOCKER_REGISTRY_PASS` in the `compose.yml` during installation, enter `DOCKER_REGISTRY_PASS` here. Do not enter the password directly.)

**[Git Auth (Git Repository)]**
Configure this if the monitored repository is a private repository.
- **Username**: Git username
- **Password Env Var**: The name of the environment variable in the Stower container where the password or PAT (Personal Access Token) is stored. (e.g., `MY_GIT_TOKEN`)

Once configured, click "Save Project".

### 3. Execute Builds and View Logs

- **Automatic Execution**: If the trigger is set to `polling`, it checks for the latest commits in the Git repository at the specified interval. When a new commit is detected, the build & push pipeline starts automatically.
- **Manual Execution**: Click the ▶️ (Run Build) button on each project card to run the pipeline immediately.
- **View Logs**: Click the 📄 (View Logs) button to check the real-time logs of the currently running build or the last executed build.

### 4. Under the Hood of the Pipeline

When Stower executes a project, the following operations are automatically performed in the background:

1. `docker login` (if registry information is configured)
2. `git clone` (fetches the latest code from the specified branch)
3. `docker compose build` (builds the container image)
4. `docker compose push` (pushes to the registry)

## 🛠 Stack

- **Backend**: Go 1.22
- **Frontend**: Vanilla JS / HTML / CSS
- **Infrastructure**: Docker, Docker Compose

---

<br>

# 🇯🇵 日本語 (Japanese)

# 🚢 Stower CI

Stower（ストワー）は、軽量でセルフホスト可能なDocker ComposeネイティブのCI/CDツールです。
対象のリポジトリを監視し、変更を検知して自動的に `docker compose build` と `docker compose push` を実行します。直感的なWeb UIを備えており、ビルドログのリアルタイム確認や手動実行も可能です。

## ✨ 特徴

- **Docker Composeネイティブ**: 監視対象リポジトリの `compose.yml` をそのままビルド手順として利用します。
- **シンプルなWeb UI**: プロジェクトの管理、ステータス確認、リアルタイムなログ監視がブラウザ上で完結します。
- **柔軟なトリガー**: Gitリポジトリのポーリング（定期監視）と、手動トリガーに対応。
- **安全な認証管理**: Gitのプライベートリポジトリや、プライベートなDockerコンテナレジストリへの認証情報を環境変数経由で安全に扱います。

---

## 🚀 インストール & 起動

Stower本体はDockerコンテナとして提供されています。本番環境（またはローカル環境）で以下の `compose.yml` を配置し、起動してください。

```yaml
services:
  stower:
    image: ghcr.io/<YOUR_GITHUB_USERNAME>/stower:latest
    container_name: stower-ci
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./stower-data:/app/workspace
    environment:
      - TZ=Asia/Tokyo
      # 例: プライベートリポジトリ用のトークンやレジストリのパスワードを設定
      # - MY_GIT_TOKEN=ghp_xxxxxxxxx
      # - DOCKER_REGISTRY_PASS=xxxxxxxxx
```

```bash
docker compose pull
docker compose up -d
```

起動後、ブラウザで `http://<サーバーのIP>:8080` にアクセスするとダッシュボードが表示されます。

## 📖 使い方

Stowerを使ってプロジェクトを自動ビルド・プッシュするための手順を解説します。

### 1. 監視対象リポジトリの準備

Stowerは、対象リポジトリのルートディレクトリにある `compose.yml` を使ってビルドとプッシュを行います。そのため、リポジトリ内の `compose.yml` には `build` と `image` の両方を定義しておく必要があります。

例：監視対象リポジトリの `compose.yml`

```yaml
services:
  web:
    # Stowerが実行する `docker compose build` の対象
    build: .
    # Stowerがビルド後に `docker compose push` する先（レジストリURL/イメージ名:タグ）
    image: localhost:5000/test-web:latest
    ports:
      - "8080:80"
```
※ `image:` には、プッシュ先のレジストリのアドレスを必ず含めてください（Docker Hubの場合は省略可能です）。

### 2. プロジェクトの追加（Web UI）

Stowerのダッシュボード右上にある 「New Project」 をクリックし、監視したいプロジェクトを登録します。

**基本設定**
- **Repository URL**: 監視対象のGitリポジトリURL（例: `https://github.com/user/repo.git`）
- **Branch**: 監視対象のブランチ名（例: `main`）

**Trigger（実行条件）の設定**
- **Type**: `polling` (定期監視) を選択します。
- **Interval**: 監視間隔を指定します。（例: `1m` = 1分、`30s` = 30秒）

**🔐 認証情報の設定（オプション）**

**【Registry (Docker レジストリ)】**
プッシュ先がプライベートレジストリの場合に設定します。
- **Server**: レジストリのサーバーアドレス（例: `ghcr.io` や `localhost:5000`）
- **Username**: レジストリのユーザー名
- **Password Env Var**: パスワードやトークンが格納されている Stower本体の環境変数名。（例: 上記インストール手順の `compose.yml` で `DOCKER_REGISTRY_PASS` と設定した場合は、ここに `DOCKER_REGISTRY_PASS` と入力します。直接パスワードは入力しません）

**【Git Auth (Git リポジトリ)】**
監視対象がプライベートリポジトリの場合に設定します。
- **Username**: Gitのユーザー名
- **Password Env Var**: パスワードやPAT（Personal Access Token）が格納されている Stower本体の環境変数名。（例: `MY_GIT_TOKEN`）

設定が完了したら 「Save Project」 をクリックします。

### 3. ビルドの実行とログ確認

- **自動実行**: トリガーを `polling` に設定した場合、指定した間隔でGitリポジトリの最新コミットをチェックします。新しいコミットが検知されると、自動的にビルド＆プッシュのパイプラインが開始されます。
- **手動実行**: 各プロジェクトカードの ▶️（Run Build） ボタンをクリックすると、即座にパイプラインを実行できます。
- **ログの確認**: 📄（View Logs） ボタンをクリックすると、現在進行中のビルド、または最後に実行されたビルドのリアルタイムログを確認できます。

### 4. パイプラインの裏側の挙動

Stowerがプロジェクトを実行する際、裏側では以下の操作が自動で行われます。

1. `docker login` (レジストリ情報が設定されている場合)
2. `git clone` (指定したブランチの最新コードを取得)
3. `docker compose build` (コンテナイメージのビルド)
4. `docker compose push` (レジストリへのプッシュ)

## 🛠 構成

- **Backend**: Go 1.22
- **Frontend**: Vanilla JS / HTML / CSS
- **Infrastructure**: Docker, Docker Compose
