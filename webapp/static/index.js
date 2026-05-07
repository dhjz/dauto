const { createApp } = Vue

const baseUrl = window.location.origin

function apiFetch(url, options = {}) {
  const token = localStorage.getItem('token')
  if (token) {
    options.headers = options.headers || {}
    options.headers['X-Token'] = token
  }
  return fetch(url.startsWith('http') ? url : baseUrl + url, options)
}

var app = createApp({
  data() {
    return {
      activeTab: 'dashboard',
      token: localStorage.getItem('token') || '',
      isLoggedIn: !!localStorage.getItem('token'),
      showLoginModal: !localStorage.getItem('token'),
      loginForm: {
        password: ''
      },
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
      executionOffset: 0,
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
      executionTimer: null,
      showAddLogModal: false,
      newLogFilePath: '',
      logFiles: [],
      selectedLogFile: '',
      logContent: '',
      logElement: null,
      logTailLines: 5000,
      autoScroll: true,
      autoWrapLog: true,
      logEventSource: null
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
    this.loadLogFiles()
    this.refreshTimer = setInterval(() => {
      if (this.isLoggedIn && this.activeTab === 'executions') {
        this.loadExecutions()
      }
    }, 3000)
  },
  beforeUnmount() {
    if (this.refreshTimer) clearInterval(this.refreshTimer)
    this.clearExecutionTimer()
    this.stopLogStream()
  },
  methods: {
    clearExecutionTimer() {
      if (this.executionTimer) clearInterval(this.executionTimer)
      this.executionTimer = null
    },
    async login() {
      const res = await fetch(baseUrl + '/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password: this.loginForm.password })
      })
      if (res.ok) {
        const data = await res.json()
        if (data.token) {
          this.token = data.token
          localStorage.setItem('token', data.token)
          this.isLoggedIn = true
          this.showLoginModal = false
          this.loginForm.password = ''
          this.loadData()
        }
      } else {
        alert('密码错误')
      }
    },
    logout() {
      if (!confirm('确定退出登录吗？')) return
      this.token = ''
      localStorage.removeItem('token')
      this.isLoggedIn = false
      this.projects = []
      this.tasks = []
      this.executions = []
    },
    async checkLogin() {
      if (!this.isLoggedIn) {
        this.showLoginModal = true
        return false
      }
      return true
    },
    async loadData() {
      if (!this.isLoggedIn) return
      await Promise.all([
        this.loadConfig(),
        this.loadProjects(),
        this.loadTasks(),
        this.loadExecutions()
      ]).catch((err) => {
        if (err.message && err.message.includes('401')) {
          this.showLoginModal = true
        }
      })
    },
    async loadConfig() {
      try {
        const res = await apiFetch('/api/config')
        const data = await res.json()
        if (data && Object.keys(data).length > 0) {
          this.config = { ...this.config, ...data }
        }
      } catch (e) {
        console.error('加载配置失败', e)
      }
    },
    async saveConfig() {
      await apiFetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(this.config)
      })
      alert('配置已保存')
    },
    async loadProjects() {
      try {
        const res = await apiFetch('/api/projects')
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
        const res = await apiFetch('/api/tasks')
        this.tasks = await res.json()
      } catch (e) {
        console.error('加载任务失败', e)
        this.tasks = []
      }
    },
    async loadExecutions(force = false) {
      try {
        const url = `/api/executions?limit=${force === true ? '': (this.executionOffset || '')}`
        const res = await apiFetch(url)
        const newExecutions = await res.json()
        if (force === true) {
          this.executions = newExecutions
        } else if (this.executionOffset > 0 && newExecutions.length > 0) {
          this.executions = [...this.executions, ...newExecutions]
        } else if (this.executionOffset === 0) {
          this.executions = newExecutions
        }
        this.executionOffset = this.executions.length
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
      if (!this.isLoggedIn) return
      try {
        const res = await apiFetch('/api/environments')
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
      let url = '/api/projects'
      let method = 'POST'
      if (this.editingProject) {
        url = '/api/projects/' + this.editingProject.id
        method = 'PUT'
      }
      await apiFetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      })
      this.showProjectModal = false
      this.loadProjects()
    },
    async deleteProject(id) {
      if (confirm('确定删除该项目?')) {
        await apiFetch('/api/projects/' + id, { method: 'DELETE' })
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
      let url = '/api/tasks'
      let method = 'POST'
      if (this.editingTask) {
        url = '/api/tasks/' + this.editingTask.id
        method = 'PUT'
      }
      await apiFetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      })
      this.showTaskModal = false
      this.loadTasks()
    },
    async deleteTask(id) {
      if (confirm('确定删除该任务?')) {
        await apiFetch('/api/tasks/' + id, { method: 'DELETE' })
        this.loadTasks()
      }
    },
    async runProject(projectId, force = false, module = '') {
      this.loading = true
      const res = await apiFetch('/api/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ projectId, force, module })
      })
      const data = await res.json()
      if (res.ok && data.id) {
        this.executionDetail = { id: data.id, projectId: projectId, status: 'running', startTime: Date.now(), output: '' }
        this.showExecutionModal = true
        this.clearExecutionTimer()
        this.executionTimer = setInterval(() => {
          this.loadExecutionDetail(data.id)
        }, 2000)
      } else if (data.error) {
        alert(data.error)
      }
      setTimeout(() => { this.loading = false }, 2000)
    },
    async loadExecutionDetail(id) {
      try {
        const res = await apiFetch('/api/executions/' + id)
        if (res.ok) {
          this.executionDetail = await res.json()
          if (this.executionDetail.status !== 'running') {
            this.clearExecutionTimer()
          }
        }
      } catch (e) {
        console.error('加载执行详情失败', e)
      }
    },
    async toggleTask(task) {
      const updated = { ...task, enabled: !task.enabled }
      await apiFetch('/api/tasks/' + task.id, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updated)
      })
      this.loadTasks()
    },
    openModuleSelect(projectId, force) {
      const project = this.projects.find(p => p.id === projectId)
      if (!project || !project.modules || project.modules.length === 0) {
        if (!confirm('确定要执行该项目吗？')) return
        this.runProject(projectId, force, '')
        return
      }
      this.selectedProjectForModule = projectId
      this.selectedModule = ''
      this.selectedModuleForce = force
      this.showModuleSelectModal = true
    },
    confirmRunModule() {
      if (!confirm('确定要执行该项目吗？')) return
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
        this.clearExecutionTimer()
        this.executionTimer = setInterval(() => {
          this.loadExecutionDetail(execution.id)
        }, 1000)
      }
    },
    closeExecutionModal() {
      this.showExecutionModal = false
      this.clearExecutionTimer()
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
    },
    loadLogFiles() {
      const files = localStorage.getItem('logFiles')
      this.logFiles = files ? JSON.parse(files) : []
      const tailLines = localStorage.getItem('logTailLines')
      if (tailLines) {
        this.logTailLines = parseInt(tailLines) || 5000
      }
      const autoWrap = localStorage.getItem('autoWrapLog')
      this.autoWrapLog = autoWrap !== null ? autoWrap === 'true' : true
    },
    addLogFile() {
      if (!this.newLogFilePath.trim()) {
        alert('请输入文件路径')
        return
      }
      if (!this.logFiles.includes(this.newLogFilePath)) {
        this.logFiles.push(this.newLogFilePath)
        localStorage.setItem('logFiles', JSON.stringify(this.logFiles))
      }
      this.selectedLogFile = this.newLogFilePath
      this.showAddLogModal = false
      this.newLogFilePath = ''
      this.startLogStream()
    },
    startLogStream() {
      if (!this.selectedLogFile) {
        if (this.$refs.logContent) {
          this.$refs.logContent.innerHTML = ''
        }
        this.stopLogStream()
        return
      }
      this.stopLogStream()
      const token = localStorage.getItem('token')
      const tokenParam = token ? '&token=' + token : ''

      this.logElement = this.$refs.logContent
      if (this.logElement) {
        this.logElement.innerHTML = ''
      }

      fetch(`${baseUrl}/api/log/tail?path=${encodeURIComponent(this.selectedLogFile)}&lines=${this.logTailLines}${tokenParam}`)
        .then(res => res.json())
        .then(data => {
          if (data.content && this.logElement) {
            const lines = data.content.split('\n')
            const fragment = document.createDocumentFragment()
            lines.forEach(line => {
              if (line.trim()) {
                const div = document.createElement('div')
                div.textContent = line
                fragment.appendChild(div)
              }
            })
            this.logElement.appendChild(fragment)
            if (this.autoScroll) {
              this.logElement.scrollTop = this.logElement.scrollHeight
            }
          }
        })
        .catch(err => {
          console.error('加载历史日志失败', err)
        })

      const streamUrl = `${baseUrl}/api/log/stream?path=${encodeURIComponent(this.selectedLogFile)}${tokenParam}`
      this.logEventSource = new EventSource(streamUrl)
      this.logEventSource.onmessage = (e) => {
        if (e.data && this.logElement) {
          const div = document.createElement('div')
          div.textContent = e.data
          this.logElement.appendChild(div)

          // const maxLines = this.maxLogLines
          // while (this.logElement.children.length > maxLines) {
          //   this.logElement.removeChild(this.logElement.firstChild)
          // }

          if (this.autoScroll) {
            this.logElement.scrollTop = this.logElement.scrollHeight
          }
        }
      }
      this.logEventSource.onerror = () => {
        console.log('日志连接断开')
      }
    },
    stopLogStream() {
      if (this.logEventSource) {
        this.logEventSource.close()
        this.logEventSource = null
      }
    },
    reloadLog() {
      localStorage.setItem('logTailLines', this.logTailLines)
      if (this.selectedLogFile) {
        this.startLogStream()
      }
    },
    copyLogPath() {
      if (this.selectedLogFile) {
        copyText(this.selectedLogFile)
        notify('路径已复制', 1)
      }
    }
  }
})

app.mount('#app')
