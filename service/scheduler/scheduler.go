package scheduler

import (
	"dauto/service/executor"
	"dauto/service/store"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	c      *cron.Cron
	mu     sync.Mutex
	jobMap = make(map[string]cron.EntryID)
)

func Start() {
	mu.Lock()
	c = cron.New(cron.WithSeconds())
	mu.Unlock()

	loadJobs()

	c.Start()
	log.Println("定时任务调度器已启动")
}

func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if c != nil {
		c.Stop()
	}
}

func loadJobs() {
	s := store.GetStore()
	for _, task := range s.Tasks {
		if task.Enabled {
			addCronJob(task)
		}
	}
}

func addCronJob(task *store.Task) {
	mu.Lock()
	defer mu.Unlock()

	if c == nil {
		return
	}

	entryID, err := c.AddFunc(task.Cron, func() {
		runTask(task)
	})
	if err != nil {
		log.Printf("添加定时任务失败: %v", err)
		return
	}
	jobMap[task.ID] = entryID
	log.Printf("添加定时任务: %s, Cron: %s", task.Name, task.Cron)
}

func RemoveJob(taskID string) {
	mu.Lock()
	defer mu.Unlock()

	if entryID, ok := jobMap[taskID]; ok {
		c.Remove(entryID)
		delete(jobMap, taskID)
	}
}

func runTask(task *store.Task) {
	log.Printf("开始执行定时任务: %s", task.Name)

	s := store.GetStore()
	var project *store.Project
	for _, p := range s.Projects {
		if p.ID == task.ProjectID {
			project = p
			break
		}
	}

	if project == nil {
		log.Printf("项目不存在: %s", task.ProjectID)
		return
	}

	if project.Running {
		log.Printf("项目 %s 正在运行中，跳过本次执行", project.Name)
		task.LastRun = time.Now().UnixMilli()
		store.UpdateTask(task)
		return
	}

	execID := fmt.Sprintf("exec-%d", time.Now().UnixMilli())
	exec := &store.Execution{
		ID:        execID,
		TaskID:    task.ID,
		ProjectID: task.ProjectID,
		Status:    "running",
		StartTime: time.Now().UnixMilli(),
	}
	store.AddExecution(exec)

	output, err := executor.RunProject(execID, task.ProjectID, false, "")

	exec = store.GetExecution(execID)
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

	task.LastRun = time.Now().UnixMilli()
	store.UpdateTask(task)

	log.Printf("定时任务执行完成: %s, Status: %s", task.Name, exec.Status)
}

func AddTask(task *store.Task) {
	if task.Enabled {
		addCronJob(task)
	}
}

func UpdateTaskCron(taskID string, cronExpr string, enabled bool) {
	RemoveJob(taskID)

	task := &store.Task{ID: taskID, Cron: cronExpr, Enabled: enabled}
	if enabled {
		addCronJob(task)
	}
}
