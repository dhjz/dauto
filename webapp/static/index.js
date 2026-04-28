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
        wechatWebhook: '',
        maxExecutions: 1000
      },
      projects: [],
      tasks: [],
      executions: [],
      environments: {},
      showProjectModal: false,
      showTaskModal: false,
      showExecutionModal: false,
      showModuleSelectModal: false,
      selectedProjectForModule: null,
      selectedModule: '',
      selectedModuleForce: false,
      executionDetail: {},
      editingProject: null,
      editingTask: null,
      projectForm: {
        name: '',
        type: 'backend',
        repoUrl: '',
        localDir: '',
        branch: 'master',
        buildCmd: '',
        skipIfNoChange: false,
        modules: [],
        javaHome: '',
        mavenHome: '',
        nodeHome: ''
      },
      taskForm: {
        name: '',
        projectId: '',
        cron: '0 */5 * * * *',
        enabled: true
      },
      loading: false,
      refreshTimer: null,
      executionTimer: null
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
    if (this.refreshTimer) clearInterval(this.refreshTimer)
    if (this.executionTimer) clearInterval(this.executionTimer)
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
        const projects = await res.json()
        projects.forEach(p => {
          p.modulesText = (p.modules || []).map(m => m.name).join(', ')
        })
        this.projects = projects
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
        if (this.showExecutionModal && this.executionDetail.id) {
          const current = this.executions.find(e => e.id === this.executionDetail.id)
          if (current) {
            this.executionDetail = current
          }
        }
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
    addModule() {
      this.projectForm.modules.push({ name: '', deployDir: '', startScript: '' })
    },
    removeModule(idx) {
      this.projectForm.modules.splice(idx, 1)
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
          skipIfNoChange: project.skipIfNoChange || false,
          deployDir: project.deployDir || '',
          modules: (project.modules || []).map(m => ({ name: m.name || '', deployDir: m.deployDir || '', startScript: m.startScript || '' })),
          javaHome: project.javaHome || '',
          mavenHome: project.mavenHome || '',
          nodeHome: project.nodeHome || ''
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
          skipIfNoChange: false,
          deployDir: '',
          modules: [],
          javaHome: '',
          mavenHome: '',
          nodeHome: ''
        }
      }
      this.showProjectModal = true
    },
    async saveProject() {
      const data = {
        name: this.projectForm.name,
        type: this.projectForm.type,
        repoUrl: this.projectForm.repoUrl,
        localDir: this.projectForm.localDir,
        branch: this.projectForm.branch,
        buildCmd: this.projectForm.buildCmd,
        skipIfNoChange: this.projectForm.skipIfNoChange,
        deployDir: this.projectForm.deployDir || '',
        modules: this.projectForm.modules.filter(m => m.name.trim()),
        javaHome: this.projectForm.javaHome,
        mavenHome: this.projectForm.mavenHome,
        nodeHome: this.projectForm.nodeHome
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
    copyProject(project) {
      this.editingProject = null
      this.projectForm = {
        name: project.name + '-copy',
        type: project.type,
        repoUrl: project.repoUrl,
        localDir: project.localDir + '-copy',
        branch: project.branch,
        buildCmd: project.buildCmd || '',
        skipIfNoChange: project.skipIfNoChange || false,
        deployDir: project.deployDir || '',
        modules: (project.modules || []).map(m => ({ name: m.name || '', deployDir: m.deployDir || '', startScript: m.startScript || '' })),
        javaHome: project.javaHome || '',
        mavenHome: project.mavenHome || '',
        nodeHome: project.nodeHome || ''
      }
      this.showProjectModal = true
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
    async runProject(projectId, force = false, module = '') {
      this.loading = true
      const res = await fetch(baseUrl + '/api/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ projectId, force, module })
      })
      const data = await res.json()
      if (res.ok && data.id) {
        this.executionDetail = { id: data.id, projectId: projectId, status: 'running', startTime: Date.now(), output: '' }
        this.showExecutionModal = true
        this.executionTimer = setInterval(() => {
          this.loadExecutions()
        }, 1000)
      } else if (data.error) {
        alert(data.error)
      }
      setTimeout(() => { this.loading = false }, 2000)
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
    openModuleSelect(projectId, force) {
      const project = this.projects.find(p => p.id === projectId)
      if (!project || !project.modules || project.modules.length === 0) {
        this.runProject(projectId, force, '')
        return
      }
      this.selectedProjectForModule = projectId
      this.selectedModule = ''
      this.selectedModuleForce = force
      this.showModuleSelectModal = true
    },
    confirmRunModule() {
      this.showModuleSelectModal = false
      this.runProject(this.selectedProjectForModule, this.selectedModuleForce, this.selectedModule || '')
    },
    getProject(projectId) {
      return this.projects.find(p => p.id === projectId) || {}
    },
    openExecutionModal(execution) {
      this.executionDetail = execution
      this.showExecutionModal = true
      if (execution.status === 'running') {
        this.executionTimer = setInterval(() => {
          this.loadExecutions()
        }, 1000)
      }
    },
    closeExecutionModal() {
      this.showExecutionModal = false
      if (this.executionTimer) {
        clearInterval(this.executionTimer)
        this.executionTimer = null
      }
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
    formatDuration(startTime, endTime) {
      if (!startTime) return '<span class="duration">-</span>'
      const end = endTime || Date.now()
      const diff = end - startTime
      let text
      if (diff < 1000) text = diff + 'ms'
      else if (diff < 60000) text = (diff / 1000).toFixed(1) + '秒'
      else {
        const minutes = Math.floor(diff / 60000)
        const seconds = ((diff % 60000) / 1000).toFixed(0)
        text = minutes + '分' + seconds + '秒'
      }
      const isLong = diff > 10000
      return '<span class="duration' + (isLong ? ' long' : '') + '">' + text + '</span>'
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
