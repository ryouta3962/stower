package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ... Docker & Git 系の関数 (runDockerCompose, loginDocker, getLatestCommit, cloneRepo) はそのまま ...

// ★ 追加: ステータスとログを安全に更新するヘルパー関数
func setProjectStatus(projID string, status string, logMsg string) {
	configMutex.Lock()
	defer configMutex.Unlock()
	for i, p := range globalConfig.Projects {
		if getProjectID(p) == projID {
			globalConfig.Projects[i].LastStatus = status
			globalConfig.Projects[i].LastLog = logMsg
			break
		}
	}
}

// ★ 修正: コマンドの出力を1行ずつリアルタイムに読み取り、コールバックに渡す関数
func runDockerComposeStream(targetDir string, logCallback func(string), args ...string) error {
	cmdArgs := append([]string{"compose"}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = targetDir

	// 標準出力と標準エラーのパイプを作成
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// 並行してログを読み取るためのWaitGroup
	var wg sync.WaitGroup
	wg.Add(2)

	readStream := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			logCallback(scanner.Text()) // 1行出力されるごとにコールバックを実行！
		}
	}

	go readStream(stdoutPipe)
	go readStream(stderrPipe)

	wg.Wait()
	return cmd.Wait()
}

func loginDocker(reg Registry) error {
	if reg.Server == "" || reg.Username == "" || reg.PasswordEnv == "" {
		return nil
	}
	pass := os.Getenv(reg.PasswordEnv)
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
// ★追加: 実際のビルド＆プッシュ処理を独立した関数に切り出し
func runPipeline(proj Project) error {
	projID := getProjectID(proj)
	dest := filepath.Join("/app/workspace", projID)

	var fullLog strings.Builder
	var logMutex sync.Mutex

	// ★ ログを追記して、即座にステータスを更新する関数
	appendLog := func(msg string) {
		logMutex.Lock()
		fullLog.WriteString(msg + "\n")
		currentLog := fullLog.String()
		logMutex.Unlock()
		
		// リアルタイムにメモリ上のログを更新
		setProjectStatus(projID, "Building", currentLog)
	}

	appendLog("Starting pipeline...")

	if err := loginDocker(proj.Registry); err != nil {
		appendLog(fmt.Sprintf("⚠️ Docker login failed (skipping): %v", err))
	}

	appendLog("📥 Cloning repository...")
	if err := cloneRepo(proj.Repo, proj.Branch, dest); err != nil {
		appendLog(fmt.Sprintf("❌ Clone failed: %v", err))
		setProjectStatus(projID, "Failed", fullLog.String())
		return err
	}

	appendLog("🔨 Starting Build...")
	// runDockerCompose ではなく、ストリーミング用の関数を呼ぶ
	err := runDockerComposeStream(dest, appendLog, "build")
	if err != nil {
		appendLog(fmt.Sprintf("❌ Build failed: %v", err))
		setProjectStatus(projID, "Failed", fullLog.String())
		return err
	}

	appendLog("🚀 Starting Push...")
	err = runDockerComposeStream(dest, appendLog, "push")
	if err != nil {
		appendLog(fmt.Sprintf("❌ Push failed: %v", err))
		setProjectStatus(projID, "Failed", fullLog.String())
		return err
	}

	appendLog("✅ Pipeline Success!")
	setProjectStatus(projID, "Success", fullLog.String())
	return nil
}

// ★追加: 手動でビルドをキックする関数（非同期で実行）
func TriggerBuild(projID string) error {
	configMutex.Lock()
	var targetProj *Project
	for _, p := range globalConfig.Projects {
		if getProjectID(p) == projID {
			targetProj = &p
			break
		}
	}
	configMutex.Unlock()

	if targetProj == nil {
		return fmt.Errorf("project not found")
	}

	// 裏側（ゴルーチン）でパイプラインを回す
	go func(p Project) {
		log.Printf("[%s] 🎯 Manual build triggered!", projID)
		if err := runPipeline(p); err != nil {
			log.Printf("[%s] ❌ Manual build failed: %v", projID, err)
		}
	}(*targetProj)

	return nil
}

// --- Polling Logic ---
func pollProject(ctx context.Context, proj Project) {
	projID := getProjectID(proj)
	duration, err := time.ParseDuration(proj.Trigger.Interval)
	if err != nil {
		log.Printf("[%s] ❌ Invalid interval: %v", projID, err)
		return
	}
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	lastHash := ""

	log.Printf("[%s] 🔄 Started polling every %s", projID, duration)

	checkAndBuild := func() {
		log.Printf("[%s] 🔍 Checking for updates...", projID)
		hash, err := getLatestCommit(proj.Repo, proj.Branch)
		if err != nil {
			log.Printf("[%s] ⚠️ Failed to fetch commit: %v", projID, err)
			return
		}
		if hash == lastHash {
			// ログが多すぎる場合はここをコメントアウトしてもOK
			// log.Printf("[%s] 💤 No changes. Waiting...", projID)
			return
		}

		log.Printf("[%s] ✨ New commit detected! %s", projID, hash[:7])
		lastHash = hash

		// 切り出した runPipeline を呼び出す
		if err := runPipeline(proj); err != nil {
			log.Printf("[%s] ❌ Pipeline failed: %v", projID, err)
		}
	}

	checkAndBuild()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] 🛑 Stopped polling.", projID)
			return
		case <-ticker.C:
			checkAndBuild()
		}
	}
}

// ゴルーチンを開始し、停止用の関数をマップに登録する
func startPolling(proj Project) {
	projID := getProjectID(proj)
	ctx, cancel := context.WithCancel(context.Background())	
	configMutex.Lock()
	cancelFuncs[projID] = cancel
	configMutex.Unlock()
	
	go pollProject(ctx, proj)
}

// ゴルーチンを停止する
func stopPolling(projID string) {
	configMutex.Lock()
	defer configMutex.Unlock()
	if cancel, exists := cancelFuncs[projID]; exists {
		cancel()
		delete(cancelFuncs, projID)
	}
}
