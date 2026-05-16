package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"gopkg.in/yaml.v3"
)

// --- 設定ファイル (config.yml) の構造体 ---
type Config struct {
	Projects []Project `yaml:"projects"`
}

type Project struct {
	Name    string  `yaml:"name"`
	Repo    string  `yaml:"repo"`
	Branch  string  `yaml:"branch"`
	Trigger Trigger `yaml:"trigger"`
}

type Trigger struct {
	Type     string `yaml:"type"`
	Interval string `yaml:"interval"` // ポーリング間隔 (例: "1m")
}

// 設定ファイルを読み込んで構造体に変換する関数
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

// ----------------------------------------

func runDockerCompose(targetDir string, args ...string) error {
	cmdArgs := append([]string{"compose"}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	fmt.Println("Starting Stower CI...")

	// --- Phase 2: 設定の読み込みテスト ---
	// コンテナにマウントされている config.yml を読み込む
	log.Println("Loading config.yml...")
	config, err := loadConfig("/app/config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 読み込んだ設定を画面に出力してみる
	for _, proj := range config.Projects {
		log.Printf("[Loaded Project] Name: %s, Repo: %s, Branch: %s, Trigger: %s (%s)",
			proj.Name, proj.Repo, proj.Branch, proj.Trigger.Type, proj.Trigger.Interval)
	}
	// -----------------------------------

	// 常駐（待機）処理
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Println("Waiting for triggers... (Press Ctrl+C to exit)")
	<-sigs

	log.Println("Shutting down Stower CI...")
}
