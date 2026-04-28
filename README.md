# DAuto 自动化部署系统

一个轻量级的自动化部署系统，支持 Java/Maven 后端项目和 Node/NPM 前端项目的自动构建与部署。

## 主要功能

- **项目管理**：支持添加、编辑、删除和复制部署项目
- **多模块构建**：后端项目支持多模块构建，可选择单独部署某个模块
- **定时任务**：支持 Cron 表达式配置定时构建任务
- **执行记录**：完整的构建日志记录，可查看每次执行的详细输出
- **运行状态检测**：防止同一项目重复执行
- **企业微信通知**：构建完成后自动推送通知
- **环境检测**：自动检测 Java、Maven、Node 环境

## 目录结构

| 目录 | 说明 |
|-----|------|
| service/executor | 构建执行器，处理 Git 拉取、编译、部署逻辑 |
| service/router | HTTP 路由处理，提供 RESTful API |
| service/scheduler | 定时任务调度器 |
| service/store | 数据存储层，使用本地 JSON 文件持久化 |
| service/utils | 工具函数 |
| webapp | Vue3 前端页面 |
| webapp/static | 前端静态资源（JS/CSS/Vue） |
| data | 数据存储目录（程序启动时自动创建） |
| dist | 编译输出的可执行文件目录 |

## 快速开始

### Windows

```bash
# 编译
go build -o dist/dauto.exe .

# 运行
dist/dauto.exe
# 或指定端口
dist/dauto.exe -p 8080
```

### Linux

```bash
# 编译
go build -o dauto .

# 运行
./dauto
# 或指定端口
./dauto -p 8080
```

服务启动后访问 http://localhost:8002

## 数据目录

程序会在可执行文件同级目录下创建 `data` 目录，包含以下文件：
- `config.json` - 系统配置
- `projects.json` - 项目配置
- `tasks.json` - 定时任务配置
- `executions.json` - 执行记录
