#!/bin/bash
set -euo pipefail
exec 9>/data/lock/autogit.lock
flock -n 9 || exit 0

# ==========================================
# 📌 配置区域（按需修改）
# ==========================================
force="${1:-}"

# --- 通用配置 ---
LOG_FILE="/data/autogit.log"
BRANCH="master"  # 默认分支
# 企业微信 Webhook 地址
WECHAT_WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key="

# --- 后端配置 ---
BACKEND_REPO_URL="https://cqaivm.860001.xyz:12239/zj-aigc/zj-aigc-mini-back.git"
BACKEND_LOCAL_DIR="/data/repos/zj-aigc-mini-back"
# Maven 命令（确保 PATH 包含 mvn）
MAVEN_CMD="mvn"
# Java 子模块数组
MODULES=("zj-aigc-mini-admin" "zj-aigc-mini-job" "zj-aigc-mini-client")
# JAR 存放根目录
JAR_DEPLOY_BASE="/data/zjaigc/backend"

# --- 前端配置 ---
FRONTEND_REPO_URL="https://cqaivm.860001.xyz:12239/zj-aigc/zj-aigc-manage-web.git"
FRONTEND_LOCAL_DIR="/data/repos/zj-aigc-manage-web"
NPM_CMD="npm"
# 前端 dist 部署目录
DIST_DEPLOY_DIR="/data/zjaigc/frontend/zj-aigc-manage-web/dist"

# ==========================================
# 🛠️ 函数定义
# ==========================================

log() {
    local msg="[$(date '+%F %T')] $1"
    echo "$msg" | tee -a "$LOG_FILE"
}

send_wechat_notification() {
    local content="$1"
    if [[ -z "$WECHAT_WEBHOOK_URL" ]]; then
        log "⚠️ Webhook 地址未配置，跳过通知"
        return
    fi
    curl -s "$WECHAT_WEBHOOK_URL" -H "Content-Type: application/json" -d "{\"msgtype\": \"text\", \"text\": {\"content\": \"$content\"}}"
}

# 初始化仓库（克隆或切换分支）
init_repo() {
    local url="$1"
    local dir="$2"
    if [[ ! -d "$dir/.git" ]]; then
        log "📥 克隆仓库: $url -> $dir"
        mkdir -p "$dir"
        git clone -b "$BRANCH" "$url" "$dir"
    else
        cd "$dir"
        git checkout "$BRANCH" 2>/dev/null || true
    fi
}

# 获取变更文件列表（fetch + diff）
get_changed_files() {
    local dir="$1"
    cd "$dir"
		git fetch origin "$BRANCH" --quiet
    # 强制把错误重定向到标准输出，别让错误乱跑
    git diff --name-only "HEAD..origin/$BRANCH"
}
# 更新本地代码到最新
update_repo() {
    log "开始拉取最新代码"
    local dir="$1"
    cd "$dir"
    git merge "origin/$BRANCH" --ff-only --quiet || git reset --hard "origin/$BRANCH"
}

# 判断模块是否变更
module_changed() {
    local module="$1"
    local changed_files="$2"
    echo "$changed_files" | grep -q "^${module}/"
}

# 构建 Java 模块
build_java_module() {
    local module="$1"
    local repo_dir="$2"
    log "🔨 构建模块: $module"
    send_wechat_notification "后端模块 [$module] 开始构建"
    cd "$repo_dir"
    
    # 构建命令：-pl 指定模块，-am 同时构建依赖，-DskipTests 跳过测试
    if $MAVEN_CMD clean package -pl "$module" -am -DskipTests -q; then
        log "✅ 构建成功: $module"
        send_wechat_notification "后端模块 [$module] 构建成功"
        # 查找 JAR
        local jar_path
        jar_path=$(find "$repo_dir/$module/target" -maxdepth 1 -name "*.jar" -type f | head -n1)
        
        if [[ -z "$jar_path" ]]; then
            log "⚠️ 未找到 JAR 文件: $module"
            return 1
        fi
        
        # 部署目录
        local deploy_dir="$JAR_DEPLOY_BASE/$module"
        mkdir -p "$deploy_dir"
        
        # 复制 JAR
        cp "$jar_path" "$deploy_dir/"
        log "📦 复制 JAR: $jar_path -> $deploy_dir/"
        
        # 执行启动脚本
        if [[ -x "$deploy_dir/start.sh" ]]; then
            log "🚀 执行启动脚本: $deploy_dir/start.sh"
            # 注意：start.sh 应自行处理后台运行，否则可能阻塞
            bash "$deploy_dir/start.sh" restart &
        else
            log "⚠️ 启动脚本不存在或无执行权限: $deploy_dir/start.sh"
        fi
    else
        log "❌ 构建失败: $module"
				send_wechat_notification "后端模块 [$module] 构建失败"
        return 1
    fi
}

# 构建前端
build_frontend() {
    log "🔨 构建前端项目"
    send_wechat_notification "前端模块 [zj-aigc-manage-web] 开始构建"
    cd "$FRONTEND_LOCAL_DIR"
    
    if $NPM_CMD install --silent && $NPM_CMD run build; then
        log "✅ 前端构建成功"
        
        mkdir -p "$DIST_DEPLOY_DIR"
        # 复制 dist 内容
        if [[ -d "dist" ]]; then
            cp -r dist/* "$DIST_DEPLOY_DIR/"
            log "📦 复制 dist -> $DIST_DEPLOY_DIR"
        else
            log "⚠️ 未找到 dist 目录"
        fi
    else
        log "❌ 前端构建失败"
        return 1
    fi
}

# ==========================================
# 🚀 主流程
# ==========================================

main() {
    log "========== 开始自动构建检查 =========="
		log "$(java --version | head -n 1)"
		log "$(mvn --version | head -n 1)"
		log "$(npm -v)"
		
		
		log "USER=$(id -un)"
		log "PATH=$PATH"
		command -v npm || echo "npm not in PATH"
    
    # 初始化仓库
    init_repo "$BACKEND_REPO_URL" "$BACKEND_LOCAL_DIR"
    init_repo "$FRONTEND_REPO_URL" "$FRONTEND_LOCAL_DIR"
    # --- 后端处理 ---
    backend_changes=$(get_changed_files "$BACKEND_LOCAL_DIR")
    log "backend_changes len: ${#backend_changes}  || [$backend_changes] || [$force]"
    if [[ -n "$backend_changes" ]] || [[ "$force" == "1" ]]; then
        # 构建完成后更新本地代码
        update_repo "$BACKEND_LOCAL_DIR"
        log "📝 后端检测到变更文件数: $(echo "$backend_changes" | wc -l)"
        
        for module in "${MODULES[@]}"; do
            # if module_changed "$module" "$backend_changes"; then
            if module_changed "$module" "$backend_changes"; then
                log "🔍 模块 [$module] 发生变更"
                build_java_module "$module" "$BACKEND_LOCAL_DIR"
            elif [[ "$force" == "1" ]]; then
                build_java_module "$module" "$BACKEND_LOCAL_DIR"
            else
                log "ℹ️ 模块 [$module] 无变更，跳过"
            fi
        done
        
    else
        log "ℹ️ 后端无变更"
    fi
    
    # --- 前端处理 ---
    frontend_changes=$(get_changed_files "$FRONTEND_LOCAL_DIR")
    
    if [[ -n "$frontend_changes" ]] || [[ "$force" == "2" ]]; then
        log "📝 前端检测到变更，开始构建"
        update_repo "$FRONTEND_LOCAL_DIR"
        build_frontend
    else
        log "ℹ️ 前端无变更"
    fi
    
    log "========== 构建检查结束 =========="
}

main "$@"