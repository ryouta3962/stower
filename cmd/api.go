package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// --- API Handlers ---

// GET /api/projects
func handleGetProjects(w http.ResponseWriter, r *http.Request) {
	configMutex.Lock()
	defer configMutex.Unlock()

	var res []ProjectResponse
	for _, p := range globalConfig.Projects {
		res = append(res, ProjectResponse{ID: getProjectID(p), Project: p})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// POST /api/projects (追加)
func handlePostProject(w http.ResponseWriter, r *http.Request) {
	var newProj Project
	if err := json.NewDecoder(r.Body).Decode(&newProj); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	configMutex.Lock()
	globalConfig.Projects = append(globalConfig.Projects, newProj)
	if err := saveConfig(); err != nil {
		configMutex.Unlock()
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	configMutex.Unlock()

	if newProj.Trigger.Type == "polling" {
		startPolling(newProj)
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"message": "Project added!"}`)
}

// DELETE /api/projects/{id} (削除)
func handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id") // Go 1.22からの新機能！

	configMutex.Lock()
	filtered := []Project{}
	found := false
	for _, p := range globalConfig.Projects {
		if getProjectID(p) == targetID {
			found = true
		} else {
			filtered = append(filtered, p)
		}
	}

	if !found {
		configMutex.Unlock()
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	globalConfig.Projects = filtered
	if err := saveConfig(); err != nil {
		configMutex.Unlock()
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	configMutex.Unlock()

	stopPolling(targetID) // 裏のゴルーチンを止める

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Project deleted!"}`)
}

// PUT /api/projects/{id} (更新)
func handlePutProject(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")

	var updatedProj Project
	if err := json.NewDecoder(r.Body).Decode(&updatedProj); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	configMutex.Lock()
	filtered := []Project{}
	found := false
	for _, p := range globalConfig.Projects {
		if getProjectID(p) == targetID {
			found = true
		} else {
			filtered = append(filtered, p)
		}
	}

	if !found {
		configMutex.Unlock()
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// 古いものを抜いて新しいものを追加する
	globalConfig.Projects = append(filtered, updatedProj)
	if err := saveConfig(); err != nil {
		configMutex.Unlock()
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}
	configMutex.Unlock()

	// 古いポーリングを止めて、新しい設定で再起動
	stopPolling(targetID)
	if updatedProj.Trigger.Type == "polling" {
		startPolling(updatedProj)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Project updated!"}`)
}


// POST /api/projects/{id}/trigger
func handleTriggerProject(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")

	if err := TriggerBuild(targetID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// 202 Accepted: 処理は受け付けた（完了は待たない）という意味のステータスコード
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, `{"message": "Build triggered successfully!"}`)
}

// GET /api/projects/{id}/logs
func handleGetProjectLogs(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")

	configMutex.Lock()
	defer configMutex.Unlock()

	for _, p := range globalConfig.Projects {
		if getProjectID(p) == targetID {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte(p.LastLog))
			return
		}
	}
	http.Error(w, "Project not found", http.StatusNotFound)
}