package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"gopkg.in/yaml.v3"
)

// --- Structs ---
type Config struct {
	Projects []Project `yaml:"projects" json:"projects"`
}

type Registry struct {
	Server      string `yaml:"server" json:"server"`
	Username    string `yaml:"username" json:"username"`
	PasswordEnv string `yaml:"password_env" json:"password_env"`
}

type GitAuth struct {
	Username    string `yaml:"username" json:"username"`
	PasswordEnv string `yaml:"password_env" json:"password_env"`
}

type Project struct {
	Repo     string   `yaml:"repo" json:"repo"`
	Branch   string   `yaml:"branch" json:"branch"`
	Trigger  Trigger  `yaml:"trigger" json:"trigger"`
	Registry Registry `yaml:"registry" json:"registry"`
	GitAuth  GitAuth  `yaml:"git_auth" json:"git_auth"`
	
	// UI表示用のステータス（YAMLには保存せず、JSONにのみ含める）
	LastStatus string `yaml:"-" json:"last_status"`
	LastLog    string `yaml:"-" json:"last_log"`
}

type Trigger struct {
	Type     string `yaml:"type" json:"type"`
	Interval string `yaml:"interval" json:"interval"`
}

// APIレスポンス用にIDを付与する構造体
type ProjectResponse struct {
	ID string `json:"id"`
	Project
}

// --- Globals ---
var (
	globalConfig *Config
	configMutex  sync.Mutex
	configPath   = "/app/workspace/config.yml"
	// ゴルーチンを停止するためのキャンセル関数を保持するマップ
	cancelFuncs  = make(map[string]context.CancelFunc)
)

// --- Helpers ---
func getProjectID(p Project) string {
	parts := strings.Split(p.Repo, "/")
	repoName := parts[len(parts)-1]
	repoName = strings.TrimSuffix(repoName, ".git")
	return fmt.Sprintf("%s-%s", repoName, p.Branch)
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

func saveConfig() error {
	data, err := yaml.Marshal(globalConfig)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func main() {
	fmt.Println("Starting Stower CI...")

	// config.ymlが存在しない場合、空のファイルを作成する
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Println("config.yml not found. Creating a new one in workspace...")
		// 完全な空ファイルでも良いですが、YAMLとしてパース可能な初期構造を入れておくと安全です
		if err := os.WriteFile(configPath, []byte("projects: []\n"), 0644); err != nil {
			log.Fatalf("Failed to create config.yml: %v", err)
		}
	}

	var err error
	globalConfig, err = loadConfig(configPath)
	if err != nil {
		globalConfig = &Config{}
	}

	for _, proj := range globalConfig.Projects {
		if proj.Trigger.Type == "polling" {
			startPolling(proj)
		}
	}

	// --- Go 1.22 の強力なメソッドルーティング ---
	http.HandleFunc("GET /api/projects", handleGetProjects)
	http.HandleFunc("POST /api/projects", handlePostProject)
	http.HandleFunc("PUT /api/projects/{id}", handlePutProject)
	http.HandleFunc("DELETE /api/projects/{id}", handleDeleteProject)
	http.HandleFunc("POST /api/projects/{id}/trigger", handleTriggerProject)
	http.HandleFunc("GET /api/projects/{id}/logs", handleGetProjectLogs)
	
	http.Handle("/", http.FileServer(http.Dir("/app/public")))

	go func() {
		log.Println("API & UI Server listening on port 8080...")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Println("Stower is up and running. (Press Ctrl+C to exit)")
	<-sigs

	log.Println("Shutting down Stower CI...")
}