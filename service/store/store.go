package store

import (
	"encoding/json"
	"os"
	"sync"
)

var (
	dataDir        = "data"
	configFile     = "data/config.json"
	projectsFile   = "data/projects.json"
	tasksFile      = "data/tasks.json"
	executionsFile = "data/executions.json"
	mu             sync.RWMutex
)

type Config struct {
	Port          int    `json:"port"`
	CacheDir      string `json:"cacheDir"`
	DeployDir     string `json:"deployDir"`
	JavaHome      string `json:"javaHome"`
	MavenHome     string `json:"mavenHome"`
	NodeHome      string `json:"nodeHome"`
	WechatWebhook string `json:"wechatWebhook"`
	MaxExecutions int    `json:"maxExecutions"`
}

type Module struct {
	Name        string `json:"name"`
	DeployDir   string `json:"deployDir"`
	StartScript string `json:"startScript"`
}

type Project struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Type           string   `json:"type"` // backend, frontend
	RepoURL        string   `json:"repoUrl"`
	LocalDir       string   `json:"localDir"`
	Branch         string   `json:"branch"`
	BuildCmd       string   `json:"buildCmd"`
	SkipIfNoChange bool     `json:"skipIfNoChange"`
	DeployDir      string   `json:"deployDir"` // 前端部署目录
	Running        bool     `json:"running"`   // 是否正在运行
	Modules        []Module `json:"modules"`
	JavaHome       string   `json:"javaHome"`
	MavenHome      string   `json:"mavenHome"`
	NodeHome       string   `json:"nodeHome"`
}

type Task struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Cron      string `json:"cron"` // cron表达式
	Enabled   bool   `json:"enabled"`
	LastRun   int64  `json:"lastRun"`
	NextRun   int64  `json:"nextRun"`
}

type Execution struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	ProjectID string `json:"projectId"`
	Status    string `json:"status"` // pending, running, success, failed
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	Log       string `json:"log"`
	Output    string `json:"output"`
}

type Store struct {
	Config     *Config      `json:"config"`
	Projects   []*Project   `json:"projects"`
	Tasks      []*Task      `json:"tasks"`
	Executions []*Execution `json:"executions"`
}

var globalStore = &Store{
	Config:     &Config{Port: 8002, CacheDir: "cache", DeployDir: "deploy", MaxExecutions: 1000},
	Projects:   make([]*Project, 0),
	Tasks:      make([]*Task, 0),
	Executions: make([]*Execution, 0),
}

func Init() error {
	os.MkdirAll(dataDir, 0755)
	loadConfig()
	loadProjects()
	loadTasks()
	loadExecutions()
	resetAllProjectsRunning()
	return nil
}

func resetAllProjectsRunning() {
	mu.Lock()
	defer mu.Unlock()
	for _, p := range globalStore.Projects {
		if p.Running {
			p.Running = false
		}
	}
	if len(globalStore.Projects) > 0 {
		data, _ := json.MarshalIndent(globalStore.Projects, "", "  ")
		os.WriteFile(projectsFile, data, 0644)
	}
}

func GetStore() *Store {
	mu.RLock()
	defer mu.RUnlock()
	return globalStore
}

func SaveConfig() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(globalStore.Config, "", "  ")
	os.WriteFile(configFile, data, 0644)
}

func SaveProjects() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(globalStore.Projects, "", "  ")
	os.WriteFile(projectsFile, data, 0644)
}

func SaveTasks() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(globalStore.Tasks, "", "  ")
	os.WriteFile(tasksFile, data, 0644)
}

func SaveExecutions() {
	mu.Lock()
	defer mu.Unlock()
	data, _ := json.MarshalIndent(globalStore.Executions, "", "  ")
	os.WriteFile(executionsFile, data, 0644)
}

func loadConfig() {
	data, err := os.ReadFile(configFile)
	if err == nil {
		json.Unmarshal(data, &globalStore.Config)
	}
	if globalStore.Config == nil {
		globalStore.Config = &Config{Port: 8002, CacheDir: "cache", DeployDir: "deploy"}
	}
}

func loadProjects() {
	data, err := os.ReadFile(projectsFile)
	if err == nil {
		json.Unmarshal(data, &globalStore.Projects)
	}
}

func loadTasks() {
	data, err := os.ReadFile(tasksFile)
	if err == nil {
		json.Unmarshal(data, &globalStore.Tasks)
	}
}

func loadExecutions() {
	data, err := os.ReadFile(executionsFile)
	if err == nil {
		json.Unmarshal(data, &globalStore.Executions)
	}
}

func AddProject(p *Project) {
	mu.Lock()
	defer mu.Unlock()
	globalStore.Projects = append(globalStore.Projects, p)
	data, _ := json.MarshalIndent(globalStore.Projects, "", "  ")
	os.WriteFile(projectsFile, data, 0644)
}

func UpdateProject(p *Project) {
	mu.Lock()
	defer mu.Unlock()
	for i, proj := range globalStore.Projects {
		if proj.ID == p.ID {
			globalStore.Projects[i] = p
			break
		}
	}
	data, _ := json.MarshalIndent(globalStore.Projects, "", "  ")
	os.WriteFile(projectsFile, data, 0644)
}

func DeleteProject(id string) {
	mu.Lock()
	defer mu.Unlock()
	var newProjects []*Project
	for _, p := range globalStore.Projects {
		if p.ID != id {
			newProjects = append(newProjects, p)
		}
	}
	globalStore.Projects = newProjects
	data, _ := json.MarshalIndent(globalStore.Projects, "", "  ")
	os.WriteFile(projectsFile, data, 0644)
}

func SetProjectRunning(id string, running bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, p := range globalStore.Projects {
		if p.ID == id {
			p.Running = running
			break
		}
	}
	data, _ := json.MarshalIndent(globalStore.Projects, "", "  ")
	os.WriteFile(projectsFile, data, 0644)
}

func AddTask(t *Task) {
	mu.Lock()
	defer mu.Unlock()
	globalStore.Tasks = append(globalStore.Tasks, t)
	data, _ := json.MarshalIndent(globalStore.Tasks, "", "  ")
	os.WriteFile(tasksFile, data, 0644)
}

func UpdateTask(t *Task) {
	mu.Lock()
	defer mu.Unlock()
	for i, task := range globalStore.Tasks {
		if task.ID == t.ID {
			globalStore.Tasks[i] = t
			break
		}
	}
	data, _ := json.MarshalIndent(globalStore.Tasks, "", "  ")
	os.WriteFile(tasksFile, data, 0644)
}

func DeleteTask(id string) {
	mu.Lock()
	defer mu.Unlock()
	var newTasks []*Task
	for _, t := range globalStore.Tasks {
		if t.ID != id {
			newTasks = append(newTasks, t)
		}
	}
	globalStore.Tasks = newTasks
	data, _ := json.MarshalIndent(globalStore.Tasks, "", "  ")
	os.WriteFile(tasksFile, data, 0644)
}

func AddExecution(e *Execution) {
	mu.Lock()
	defer mu.Unlock()
	globalStore.Executions = append(globalStore.Executions, e)
	maxExecutions := globalStore.Config.MaxExecutions
	if maxExecutions <= 0 {
		maxExecutions = 1000
	}
	if len(globalStore.Executions) > maxExecutions {
		globalStore.Executions = globalStore.Executions[len(globalStore.Executions)-maxExecutions:]
	}
	data, _ := json.MarshalIndent(globalStore.Executions, "", "  ")
	os.WriteFile(executionsFile, data, 0644)
}

func UpdateExecution(e *Execution) {
	mu.Lock()
	defer mu.Unlock()
	for i, exec := range globalStore.Executions {
		if exec.ID == e.ID {
			globalStore.Executions[i] = e
			break
		}
	}
	data, _ := json.MarshalIndent(globalStore.Executions, "", "  ")
	os.WriteFile(executionsFile, data, 0644)
}

func GetExecution(id string) *Execution {
	mu.RLock()
	defer mu.RUnlock()
	for _, exec := range globalStore.Executions {
		if exec.ID == id {
			return exec
		}
	}
	return nil
}
