# 项目技术
- 后端：Go 1.25
- 前端：Vue 3
- 数据库：本地json
- 打包脚本：build.bat, 打包后windows可执行exe文件在dist目录下
- 开发环境: windows

# 前段代码规范
- vue3使用选项式写法
- style标签里面的css代码使用紧凑模式，一个class类的所有属性占用一行不换行, 比如body { margin: 0; padding: 0; },每个class类独立占用一行
- 工具类可以写到utils.js文件中
- 颜色(借鉴element-ui): 主蓝色使用#409EFF, 成功使用#67C23A, 警告色使用#E6A23C, 危险色使用#F56C6C, 消息使用#909399

# 后端代码规范
- 数据库使用本地json文件
- 工具类可以写到utils.go文件中