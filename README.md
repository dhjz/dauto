# DAuto 自动化部署系统

一个轻量级的自动化部署系统，支持 Java/Maven 后端项目和 Node/NPM 前端项目的自动构建与部署。

## 主要功能

- **项目管理**：支持添加、编辑、删除和复制部署项目
- **多模块构建**：后端项目支持多模块构建，可选择单独部署某个模块
- **模块级变化检测**：自动检测代码变化，只构建和部署有变化的模块
- **定时任务**：支持 Cron 表达式配置定时构建任务
- **执行记录**：完整的构建日志记录，可查看每次执行的详细输出
- **运行状态检测**：防止同一项目重复执行
- **企业微信通知**：构建完成后自动推送通知
- **环境检测**：自动检测 Java、Maven、Node 环境
- **密码保护**：支持 Token 认证保护

## 目录结构

| 目录 | 说明 |
|-----|------|
| service/executor | 构建执行器，处理 Git 拉取、编译、部署逻辑 |
| service/router | HTTP 路由处理，提供 RESTful API |
| service/scheduler | 定时任务调度器 |
| service/store | 数据存储层，使用本地 JSON 文件持久化 |
| webapp | Vue3 前端页面 |
| webapp/static | 前端静态资源（JS/CSS/Vue） |
| data | 数据存储目录（程序启动时自动创建） |
| dist | 编译输出的可执行文件目录 |

## 快速开始

### 编译

```bash
# Windows
go build -o dist/dauto.exe .

# Linux
go build -o dauto .
```

### 运行

```bash
# 默认端口 8002
./dauto.exe

# 指定端口
./dauto.exe -p 8080

# 设置密码（生成 Token）
./dauto.exe -pwd yourpassword
```

服务启动后访问 http://localhost:8002

### 使用密码保护

启动时设置密码后，所有 API 请求需要在请求头中携带 Token：
- 登录获取 Token：POST /api/login，body: `{"password":"yourpassword"}`
- 后续请求Header: `X-Token: <token>`

## 数据目录

程序会在可执行文件同级目录下创建 `data` 目录，包含以下文件：
- `config.json` - 系统配置
- `projects.json` - 项目配置
- `tasks.json` - 定时任务配置
- `executions.json` - 执行记录

## 配置说明

### 项目配置

- **构建命令**：支持自定义构建命令，Maven 示例 `mvn clean package -DskipTests`
- **模块配置**：后端项目可配置多个模块，每个模块可设置部署目录和启动脚本
- **跳过无变化构建**：勾选后，如果代码没有变更则跳过构建

### 定时任务

支持 Cron 表达式（秒 分 时 日 月 周）：
- 每5分钟：`0 */5 * * * *`
- 每小时：`0 0 * * * *`
- 每天凌晨2点：`0 0 2 * * *`

# 其他说明
- 页面效果图见`appimg`目录
- <img src="https://gcore.jsdelivr.net/gh/dhjz/dauto@main/appimg/app1.jpg" style="width: 340px;"/>
- <img src="https://gcore.jsdelivr.net/gh/dhjz/dauto@main/appimg/app2.jpg" style="width: 340px;"/>
- <img src="https://gcore.jsdelivr.net/gh/dhjz/dauto@main/appimg/app3.jpg" style="width: 340px;"/>
- <img src="https://gcore.jsdelivr.net/gh/dhjz/dauto@main/appimg/app4.jpg" style="width: 340px;"/>


- 项目地址: [https://github.com/dhjz/dauto]( https://github.com/dhjz/dauto)  