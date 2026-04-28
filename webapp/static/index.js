const { createApp } = Vue

const baseUrl = window.location.origin

var app = createApp({
  data() {
    return {
      activeTab: 'dashboard',
      config: {
        port: 8002,
        cacheDir: 'cache',
        deployDir: 'deploy',
        javaHome: '',
        mavenHome: '',
        nodeHome: '',
        wechatWebhook: ''
      },
      projects: [],
      tasks: [],
      executions: [],
      environments: {},
      showProjectModal: false,
      showTaskModal: false,
      editingProject: null,
      editingTask: null,
      projectForm: {
        name: '',
        type: 'backend',
        repoUrl: '',
        localDir: '',
        branch: 'master',
        buildCmd: '',
        deployDir: '',
        startScript: '',
        modules: ''
      },
      taskForm: {
        name: '',
        projectId: '',
        cron: '0 */5 * * * *',
        enabled: true
      },
      loading: false,
      refreshTimer: null
    }
  },
  computed: {
    recentExecutions() {
      return this.executions.slice(-10).reverse()
    },
    runningCount() {
      return this.executions.filter(e => e.status === 'running').length
    }
  },
  mounted() {
    this.loadData()
    this.loadEnvironments()
    this.refreshTimer = setInterval(() => {
      this.loadExecutions()
    }, 3000)
  },
  beforeUnmount() {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer)
    }
  },
  methods: {
    async loadData() {
      await Promise.all([
        this.loadConfig(),
        this.loadProjects(),
        this.loadTasks(),
        this.loadExecutions()
      ])
    },
    async loadConfig() {
      try {
        const res = await fetch(baseUrl + '/api/config')
        const data = await res.json()
        if (data && Object.keys(data).length > 0) {
          this.config = { ...this.config, ...data }
        }
      } catch (e) {
        console.error('加载配置失败', e)
      }
    },
    async saveConfig() {
      await fetch(baseUrl + '/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(this.config)
      })
      alert('配置已保存')
    },
    async loadProjects() {
      try {
        const res = await fetch(baseUrl + '/api/projects')
        this.projects = await res.json()
      } catch (e) {
        console.error('加载项目失败', e)
        this.projects = []
      }
    },
    async loadTasks() {
      try {
        const res = await fetch(baseUrl + '/api/tasks')
        this.tasks = await res.json()
      } catch (e) {
        console.error('加载任务失败', e)
        this.tasks = []
      }
    },
    async loadExecutions() {
      try {
        const res = await fetch(baseUrl + '/api/executions')
        this.executions = await res.json()
      } catch (e) {
        console.error('加载执行记录失败', e)
        this.executions = []
      }
    },
    async loadEnvironments() {
      try {
        const res = await fetch(baseUrl + '/api/environments')
        this.environments = await res.json()
      } catch (e) {
        console.error('加载环境变量失败', e)
      }
    },
    openProjectModal(project = null) {
      if (project) {
        this.editingProject = project
        this.projectForm = {
          name: project.name,
          type: project.type,
          repoUrl: project.repoUrl,
          localDir: project.localDir,
          branch: project.branch,
          buildCmd: project.buildCmd || '',
          deployDir: project.deployDir || '',
          startScript: project.startScript || '',
          modules: (project.modules || []).join(', ')
        }
      } else {
        this.editingProject = null
        this.projectForm = {
          name: '',
          type: 'backend',
          repoUrl: '',
          localDir: '',
          branch: 'master',
          buildCmd: '',
          deployDir: '',
          startScript: '',
          modules: ''
        }
      }
      this.showProjectModal = true
    },
    async saveProject() {
      const data = {
        ...this.projectForm,
        modules: this.projectForm.modules.split(',').map(m => m.trim()).filter(m => m)
      }
      let url = baseUrl + '/api/projects'
      let method = 'POST'
      if (this.editingProject) {
        url = baseUrl + '/api/projects/' + this.editingProject.id
        method = 'PUT'
      }
      await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      })
      this.showProjectModal = false
      this.loadProjects()
    },
    async deleteProject(id) {
      if (confirm('确定删除该项目?')) {
        await fetch(baseUrl + '/api/projects/' + id, { method: 'DELETE' })
        this.loadProjects()
      }
    },
    openTaskModal(task = null) {
      if (task) {
        this.editingTask = task
        this.taskForm = {
          name: task.name,
          projectId: task.projectId,
          cron: task.cron,
          enabled: task.enabled
        }
      } else {
        this.editingTask = null
        this.taskForm = {
          name: '',
          projectId: '',
          cron: '0 */5 * * * *',
          enabled: true
        }
      }
      this.showTaskModal = true
    },
    async saveTask() {
      const data = { ...this.taskForm }
      let url = baseUrl + '/api/tasks'
      let method = 'POST'
      if (this.editingTask) {
        url = baseUrl + '/api/tasks/' + this.editingTask.id
        method = 'PUT'
      }
      await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      })
      this.showTaskModal = false
      this.loadTasks()
    },
    async deleteTask(id) {
      if (confirm('确定删除该任务?')) {
        await fetch(baseUrl + '/api/tasks/' + id, { method: 'DELETE' })
        this.loadTasks()
      }
    },
    async runProject(projectId) {
      this.loading = true
      await fetch(baseUrl + '/api/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ projectId })
      })
      setTimeout(() => {
        this.loadExecutions()
        this.loading = false
      }, 2000)
    },
    async toggleTask(task) {
      const updated = { ...task, enabled: !task.enabled }
      await fetch(baseUrl + '/api/tasks/' + task.id, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updated)
      })
      this.loadTasks()
    },
    getProjectName(projectId) {
      const p = this.projects.find(p => p.id === projectId)
      return p ? p.name : projectId
    },
    getProjectType(projectId) {
      const p = this.projects.find(p => p.id === projectId)
      return p ? p.type : '-'
    },
    formatTime(timestamp) {
      if (!timestamp) return '-'
      return new Date(timestamp).toLocaleString('zh-CN')
    },
    formatCron(cron) {
      const parts = cron.split(' ')
      if (parts.length === 6) {
        return `秒:${parts[0]} 分:${parts[1]} 时:${parts[2]} 日:${parts[3]} 月:${parts[4]} 周:${parts[5]}`
      }
      return cron
    },
    getStatusClass(status) {
      const map = { success: 'success', failed: 'failed', running: 'running', pending: 'pending' }
      return map[status] || ''
    }
  }
})

app.mount('#app')
