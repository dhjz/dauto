package router

import (
	"dauto/service/executor"
	"dauto/service/scheduler"
	"dauto/service/store"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

func SetupRoutesAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/config", corsHandler(handleConfig))
	mux.HandleFunc("/api/projects", corsHandler(handleProjects))
	mux.HandleFunc("/api/projects/", corsHandler(handleProjectDetail))
	mux.HandleFunc("/api/tasks", corsHandler(handleTasks))
	mux.HandleFunc("/api/tasks/", corsHandler(handleTaskDetail))
	mux.HandleFunc("/api/executions", corsHandler(handleExecutions))
	mux.HandleFunc("/api/executions/", corsHandler(handleExecutionDetail))
	mux.HandleFunc("/api/run", corsHandler(handleRunProject))
	mux.HandleFunc("/api/build", corsHandler(handleBuild))
	mux.HandleFunc("/api/environments", corsHandler(handleEnvironments))
}

func corsHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		fn(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	s := store.GetStore()
	switch r.Method {
	case "GET":
		writeJSON(w, s.Config)
	case "POST":
		var config store.Config
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		s.Config = &config
		store.SaveConfig()
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleProjects(w http.ResponseWriter, r *http.Request) {
	s := store.GetStore()
	switch r.Method {
	case "GET":
		writeJSON(w, s.Projects)
	case "POST":
		var project store.Project
		if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		project.ID = uuid.New().String()
		store.AddProject(&project)
		writeJSON(w, project)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleProjectDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/projects/"):]
	s := store.GetStore()

	var project *store.Project
	for _, p := range s.Projects {
		if p.ID == id {
			project = p
			break
		}
	}

	if project == nil {
		http.Error(w, "project not found", 404)
		return
	}

	switch r.Method {
	case "GET":
		writeJSON(w, project)
	case "PUT":
		var updated store.Project
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		updated.ID = id
		store.UpdateProject(&updated)
		writeJSON(w, updated)
	case "DELETE":
		store.DeleteProject(id)
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	s := store.GetStore()
	switch r.Method {
	case "GET":
		writeJSON(w, s.Tasks)
	case "POST":
		var task store.Task
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		task.ID = uuid.New().String()
		store.AddTask(&task)
		scheduler.AddTask(&task)
		writeJSON(w, task)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/tasks/"):]
	s := store.GetStore()

	var task *store.Task
	for _, t := range s.Tasks {
		if t.ID == id {
			task = t
			break
		}
	}

	if task == nil {
		http.Error(w, "task not found", 404)
		return
	}

	switch r.Method {
	case "GET":
		writeJSON(w, task)
	case "PUT":
		var updated store.Task
		if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		updated.ID = id
		store.UpdateTask(&updated)
		scheduler.RemoveJob(id)
		if updated.Enabled {
			scheduler.AddTask(&updated)
		}
		writeJSON(w, updated)
	case "DELETE":
		scheduler.RemoveJob(id)
		store.DeleteTask(id)
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleExecutions(w http.ResponseWriter, r *http.Request) {
	s := store.GetStore()
	switch r.Method {
	case "GET":
		writeJSON(w, s.Executions)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func handleExecutionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/executions/"):]
	s := store.GetStore()

	var execution *store.Execution
	for _, e := range s.Executions {
		if e.ID == id {
			execution = e
			break
		}
	}

	if execution == nil {
		http.Error(w, "execution not found", 404)
		return
	}

	writeJSON(w, execution)
}

func handleRunProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	var req struct {
		ProjectID string `json:"projectId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	execID := fmt.Sprintf("exec-%d", time.Now().UnixMilli())
	exec := &store.Execution{
		ID:        execID,
		TaskID:    "manual",
		ProjectID: req.ProjectID,
		Status:    "running",
		StartTime: time.Now().UnixMilli(),
	}
	store.AddExecution(exec)
	writeJSON(w, exec)

	go func() {
		output, err := executor.RunProject(execID, req.ProjectID)

		exec := store.GetExecution(execID)
		if exec == nil {
			return
		}
		exec.EndTime = time.Now().UnixMilli()
		if err != nil {
			exec.Status = "failed"
			exec.Log = err.Error()
		} else {
			exec.Status = "success"
		}
		exec.Output = output

		store.UpdateExecution(exec)

		s := store.GetStore()
		if s.Config.WechatWebhook != "" {
			msg := fmt.Sprintf("项目构建完成: %s, 状态: %s", req.ProjectID, exec.Status)
			executor.SendWechatNotification(s.Config.WechatWebhook, msg)
		}
	}()
}

func handleBuild(w http.ResponseWriter, r *http.Request) {
	handleRunProject(w, r)
}

func handleEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	envs := map[string]string{
		"java":  findCommand("java"),
		"maven": findCommand("mvn"),
		"node":  findCommand("node"),
		"npm":   findCommand("npm"),
		"git":   findCommand("git"),
		"curl":  findCommand("curl"),
	}

	writeJSON(w, envs)
}

func findCommand(name string) string {
	cmd := exec.Command("where", name)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
