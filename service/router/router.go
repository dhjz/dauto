package router

import (
	"crypto/md5"
	"dauto/service/executor"
	"dauto/service/scheduler"
	"dauto/service/store"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

var token string

func SetupRoutesAPI(mux *http.ServeMux, password string) {
	if password != "" {
		hash := md5.Sum([]byte(password))
		token = hex.EncodeToString(hash[:])
		log.Printf("Token 已启用")
	}

	mux.HandleFunc("/api/login", corsHandler(handleLogin))
	mux.HandleFunc("/api/config", corsHandler(authHandler(handleConfig)))
	mux.HandleFunc("/api/projects", corsHandler(authHandler(handleProjects)))
	mux.HandleFunc("/api/projects/", corsHandler(authHandler(handleProjectDetail)))
	mux.HandleFunc("/api/tasks", corsHandler(authHandler(handleTasks)))
	mux.HandleFunc("/api/tasks/", corsHandler(authHandler(handleTaskDetail)))
	mux.HandleFunc("/api/executions", corsHandler(authHandler(handleExecutions)))
	mux.HandleFunc("/api/executions/", corsHandler(authHandler(handleExecutionDetail)))
	mux.HandleFunc("/api/run", corsHandler(authHandler(handleRunProject)))
	mux.HandleFunc("/api/build", corsHandler(authHandler(handleBuild)))
	mux.HandleFunc("/api/environments", corsHandler(authHandler(handleEnvironments)))
	mux.HandleFunc("/api/log", corsHandler(authHandler(handleLog)))
}

func authHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			fn(w, r)
			return
		}
		reqToken := r.Header.Get("X-Token")
		if reqToken == "" {
			reqToken = r.URL.Query().Get("token")
		}
		if reqToken != token {
			http.Error(w, "Unauthorized", 401)
			return
		}
		fn(w, r)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	if token == "" {
		writeJSON(w, map[string]bool{"success": true})
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	hash := md5.Sum([]byte(req.Password))
	inputToken := hex.EncodeToString(hash[:])
	if inputToken == token {
		writeJSON(w, map[string]string{"token": token})
	} else {
		http.Error(w, "密码错误", 401)
	}
}

func corsHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Token")
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
		limit := r.URL.Query().Get("limit")
		var execs []*store.Execution
		if limit != "" {
			var n int
			fmt.Sscanf(limit, "%d", &n)
			if n >= len(s.Executions) {
				execs = []*store.Execution{}
			} else {
				execs = s.Executions[n:]
			}
		} else {
			execs = s.Executions
		}
		writeJSON(w, execs)
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
		Force     bool   `json:"force"`
		Module    string `json:"module"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	s := store.GetStore()
	var project *store.Project
	for _, p := range s.Projects {
		if p.ID == req.ProjectID {
			project = p
			break
		}
	}

	if project == nil {
		http.Error(w, "项目不存在", 404)
		return
	}

	if project.Running {
		writeJSON(w, map[string]string{"error": "项目正在运行中，请勿重复执行"})
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
		output, err := executor.RunProject(execID, req.ProjectID, req.Force, req.Module)

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

	s := store.GetStore()
	javaHome := s.Config.JavaHome
	mavenHome := s.Config.MavenHome
	nodeHome := s.Config.NodeHome

	envs := map[string]string{
		"java":  executor.GetJavaVersion(javaHome),
		"maven": executor.GetMavenVersion(mavenHome),
		"node":  executor.GetNodeVersion(nodeHome),
	}

	writeJSON(w, envs)
}

func findCommand(name string) string {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("where", name)
	} else {
		cmd = exec.Command("which", name)
	}
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

func handleLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	filePath := r.URL.Query().Get("path")
	tailLines := r.URL.Query().Get("lines")
	if filePath == "" {
		http.Error(w, "path is required", 400)
		return
	}

	if runtime.GOOS == "linux" {
		if !strings.HasPrefix(filePath, "/data/") && !strings.HasPrefix(filePath, "/opt/") {
			http.Error(w, "安全限制: Linux系统只允许监控 /data 和 /opt 目录下的文件", 403)
			return
		}
	}

	lines := 5000
	if tailLines != "" {
		fmt.Sscanf(tailLines, "%d", &lines)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", "Get-Content -Path '"+filePath+"' -Tail "+fmt.Sprintf("%d", lines)+" -Wait -Encoding UTF8")
	} else {
		cmd = exec.Command("tail", "-n", fmt.Sprintf("%d", lines), "-f", filePath)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	notify := r.Context().Done()

	go func() {
		<-notify
		cmd.Process.Kill()
	}()

	buffer := make([]byte, 1024)
	for {
		n, err := stdout.Read(buffer)
		if err != nil {
			break
		}
		if n > 0 {
			fmt.Fprintf(w, "data: %s\n\n", strings.TrimSpace(string(buffer[:n])))
			flusher.Flush()
		}
	}

	cmd.Wait()
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
