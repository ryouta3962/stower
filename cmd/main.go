package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Projects []Project `yaml:"projects"`
}

// レジストリ情報を追加
type Registry struct {
	Server      string `yaml:"server"`
	Username    string `yaml:"username"`
	PasswordEnv string `yaml:"password_env"`
}

type Project struct {
	Name     string   `yaml:"name"`
	Repo     string   `yaml:"repo"`
	Branch   string   `yaml:"branch"`
	Trigger  Trigger  `yaml:"trigger"`
	Registry Registry `yaml:"registry"` // 追加
}

type Trigger struct {
	Type     string `yaml:"type"`
	Interval string `yaml:"interval"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

func runDockerCompose(targetDir string, args ...string) error {
	cmdArgs := append([]string{"compose"}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Dockerレジストリへのログイン処理（設定がある場合のみ）
func loginDocker(reg Registry) error {
	// ユーザー名やパスワード設定がない場合はログインをスキップ（ローカルテスト用など）
	if reg.Server == "" || reg.Username == "" || reg.PasswordEnv == "" {
		return nil
	}
	pass := os.Getenv(reg.PasswordEnv)
	log.Printf("Logging into registry: %s...", reg.Server)
	cmd := exec.Command("docker", "login", reg.Server, "-u", reg.Username, "--password-stdin")
	cmd.Stdin = strings.NewReader(pass)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getLatestCommit(repo, branch string) (string, error) {
	cmd := exec.Command("git", "ls-remote", repo, branch)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(out))
	if len(fields) > 0 {
		return fields[0], nil
	}
	return "", fmt.Errorf("branch not found")
}

func cloneRepo(repo, branch, dest string) error {
	os.RemoveAll(dest)
	cmd := exec.Command("git", "clone", "--branch", branch, "--single-branch", repo, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func pollProject(proj Project) {
	duration, err := time.ParseDuration(proj.Trigger.Interval)
	if err != nil {
		log.Printf("[%s] Invalid interval %s: %v", proj.Name, proj.Trigger.Interval, err)
		return
	}
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	lastHash := ""
	dest := filepath.Join("/app/workspace", proj.Name)

	log.Printf("[%s] Started polling every %s", proj.Name, duration)

	checkAndBuild := func() {
		log.Printf("[%s] Checking for updates...", proj.Name)
		hash, err := getLatestCommit(proj.Repo, proj.Branch)
		if err != nil {
			log.Printf("[%s] Failed to fetch commit: %v", proj.Name, err)
			return
		}

		if hash == lastHash {
			return // 変更なし
		}
		log.Printf("[%s] New commit detected! %s", proj.Name, hash[:7])
		lastHash = hash

		// 1. レジストリにログイン (必要な場合)
		if err := loginDocker(proj.Registry); err != nil {
			log.Printf("[%s] Docker login failed: %v", proj.Name, err)
			return
		}

		// 2. クローン
		log.Printf("[%s] Cloning repository...", proj.Name)
		if err := cloneRepo(proj.Repo, proj.Branch, dest); err != nil {
			log.Printf("[%s] Clone failed: %v", proj.Name, err)
			return
		}

		// 3. ビルド
		log.Printf("[%s] Starting Docker Compose Build...", proj.Name)
		if err := runDockerCompose(dest, "build"); err != nil {
			log.Printf("[%s] Build failed: %v", proj.Name, err)
			return
		}

		// 4. プッシュ (追加!)
		log.Printf("[%s] Starting Docker Compose Push...", proj.Name)
		if err := runDockerCompose(dest, "push"); err != nil {
			log.Printf("[%s] Push failed: %v", proj.Name, err)
			return
		}

		log.Printf("[%s] Pipeline Success! Build & Push completed.", proj.Name)
	}

	checkAndBuild()
	for range ticker.C {
		checkAndBuild()
	}
}

func main() {
	fmt.Println("Starting Stower CI...")

	config, err := loadConfig("/app/config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	for _, proj := range config.Projects {
		if proj.Trigger.Type == "polling" {
			go pollProject(proj)
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Println("Stower is up and running. (Press Ctrl+C to exit)")
	<-sigs

	log.Println("Shutting down Stower CI...")
}
