package executor

import (
	"dauto/service/store"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func RunProject(execID string, projectID string, force bool) (string, error) {
	s := store.GetStore()
	var project *store.Project
	for _, p := range s.Projects {
		if p.ID == projectID {
			project = p
			break
		}
	}

	if project == nil {
		return "", fmt.Errorf("项目不存在: %s", projectID)
	}

	config := s.Config

	javaHome := project.JavaHome
	if javaHome == "" {
		javaHome = config.JavaHome
	}
	mavenHome := project.MavenHome
	if mavenHome == "" {
		mavenHome = config.MavenHome
	}
	nodeHome := project.NodeHome
	if nodeHome == "" {
		nodeHome = config.NodeHome
	}

	projectEnv := &store.Config{
		JavaHome:  javaHome,
		MavenHome: mavenHome,
		NodeHome:  nodeHome,
	}

	updateOutput := func(msg string) {
		exec := store.GetExecution(execID)
		if exec != nil {
			exec.Output += msg
			store.UpdateExecution(exec)
		}
	}

	updateOutput(fmt.Sprintf("开始构建项目: %s\n", project.Name))
	updateOutput(fmt.Sprintf("项目类型: %s\n", project.Type))
	updateOutput(fmt.Sprintf("仓库地址: %s\n", project.RepoURL))
	updateOutput(fmt.Sprintf("本地目录: %s\n", project.LocalDir))

	if project.Type == "backend" {
		updateOutput(fmt.Sprintf("Java: %s\n", getJavaVersion(javaHome)))
		if javaHome != "" {
			updateOutput(fmt.Sprintf("JAVA_HOME: %s\n", javaHome))
		}
		if mavenHome != "" {
			updateOutput(fmt.Sprintf("MAVEN_HOME: %s\n", mavenHome))
		}
	} else if project.Type == "frontend" {
		updateOutput(fmt.Sprintf("Node: %s\n", getNodeVersion(nodeHome)))
		if nodeHome != "" {
			updateOutput(fmt.Sprintf("NODE_HOME: %s\n", nodeHome))
		}
	}

	if err := os.MkdirAll(project.LocalDir, 0755); err != nil {
		updateOutput(fmt.Sprintf("创建目录失败: %v\n", err))
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	hasChanges := true
	if _, err := os.Stat(filepath.Join(project.LocalDir, ".git")); os.IsNotExist(err) {
		updateOutput("克隆仓库...\n")
		if err := runCommandWithOutput("", updateOutput, "git", "clone", "-b", project.Branch, project.RepoURL, project.LocalDir); err != nil {
			updateOutput(fmt.Sprintf("克隆仓库失败: %v\n", err))
			return "", fmt.Errorf("克隆仓库失败: %v", err)
		}
		hasChanges = true
	} else {
		updateOutput("更新仓库...\n")
		if err := runCommandWithOutput(project.LocalDir, updateOutput, "git", "fetch", "origin", project.Branch); err != nil {
			log.Printf("git fetch 警告: %v", err)
		}
		if err := runCommandWithOutput(project.LocalDir, updateOutput, "git", "checkout", project.Branch); err != nil {
			log.Printf("git checkout 警告: %v", err)
		}

		hasChanges = checkForChanges(project.LocalDir, project.Branch)
		updateOutput(fmt.Sprintf("代码变更检测: %v\n", hasChanges))
	}

	shouldBuild := force || !project.SkipIfNoChange || hasChanges

	if !shouldBuild && project.SkipIfNoChange {
		updateOutput("未发生变更，跳过构建\n")
		return getFinalOutput(execID), nil
	}

	if project.Type == "backend" {
		err := buildBackend(project, projectEnv, updateOutput, force)
		return getFinalOutput(execID), err
	} else if project.Type == "frontend" {
		err := buildFrontend(project, projectEnv, updateOutput, force)
		return getFinalOutput(execID), err
	}

	return getFinalOutput(execID), nil
}

func checkForChanges(localDir string, branch string) bool {
	cmd := exec.Command("git", "diff", "HEAD", fmt.Sprintf("origin/%s", branch))
	cmd.Dir = localDir
	out, err := cmd.Output()
	if err != nil {
		return true
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func getFinalOutput(execID string) string {
	exec := store.GetExecution(execID)
	if exec != nil {
		return exec.Output
	}
	return ""
}

func buildBackend(project *store.Project, config *store.Config, updateOutput func(string), force bool) error {
	updateOutput("开始构建后端项目...\n")

	env := os.Getenv("PATH")
	binDir := "bin"
	pathSep := ":"
	if runtime.GOOS == "windows" {
		binDir = "bin"
		pathSep = ";"
	}
	if config.MavenHome != "" {
		env = config.MavenHome + "/" + binDir + pathSep + env
	}
	if config.JavaHome != "" {
		env = config.JavaHome + "/" + binDir + pathSep + env
	}

	buildCmd := project.BuildCmd
	if buildCmd == "" {
		buildCmd = "mvn clean package -DskipTests"
	}

	hasModuleVar := strings.Contains(buildCmd, "{module}")

	if hasModuleVar && len(project.Modules) > 0 {
		for _, module := range project.Modules {
			moduleCmd := strings.ReplaceAll(buildCmd, "{module}", module.Name)
			updateOutput("执行构建命令: " + moduleCmd + "\n")
			if err := runCommandWithEnvAndOutput(project.LocalDir, moduleCmd, env, updateOutput); err != nil {
				updateOutput(fmt.Sprintf("模块 %s 构建失败: %v\n", module.Name, err))
				return err
			}
			updateOutput("模块 " + module.Name + " 构建成功\n")

			if module.DeployDir != "" {
				os.MkdirAll(module.DeployDir, 0755)
				jarPath := findLatestJar(project.LocalDir + "/" + module.Name + "/target")
				if jarPath != "" {
					deployPath := module.DeployDir
					os.MkdirAll(deployPath, 0755)
					destPath := filepath.Join(deployPath, filepath.Base(jarPath))
					if err := copyFile(jarPath, destPath); err != nil {
						updateOutput(fmt.Sprintf("复制JAR失败: %v\n", err))
					} else {
						updateOutput(fmt.Sprintf("部署JAR: %s -> %s\n", jarPath, destPath))
					}
				} else {
					updateOutput(fmt.Sprintf("未找到模块 %s 的JAR文件\n", module.Name))
				}

				if module.StartScript != "" {
					updateOutput(fmt.Sprintf("执行启动脚本: %s\n", module.StartScript))
					if err := runCommand(module.DeployDir, "bash", module.StartScript, "restart"); err != nil {
						log.Printf("启动脚本执行失败: %v", err)
					}
				}
			}
		}
		updateOutput("所有模块构建完成\n")
		return nil
	}

	if !force && project.SkipIfNoChange {
		updateOutput("检测Maven模块变更...\n")
		changed := false
		for _, module := range project.Modules {
			jarPath := findLatestJar(project.LocalDir + "/" + module.Name + "/target")
			if jarPath != "" {
				stat, err := os.Stat(jarPath)
				if err == nil {
					if stat.ModTime().After(time.Now().Add(-24 * time.Hour)) {
						changed = true
						updateOutput(fmt.Sprintf("模块 %s 有新构建: %s\n", module.Name, filepath.Base(jarPath)))
					}
				}
			}
		}
		if !changed {
			updateOutput("所有模块无可用构建产物，跳过构建\n")
			return nil
		}
	}

	updateOutput("执行构建命令: " + buildCmd + "\n")
	if err := runCommandWithEnvAndOutput(project.LocalDir, buildCmd, env, updateOutput); err != nil {
		updateOutput(fmt.Sprintf("构建失败: %v\n", err))
		return err
	}

	updateOutput("构建成功\n")

	for _, module := range project.Modules {
		if module.DeployDir != "" {
			os.MkdirAll(module.DeployDir, 0755)
			jarPath := findLatestJar(project.LocalDir + "/" + module.Name + "/target")
			if jarPath != "" {
				deployPath := module.DeployDir
				os.MkdirAll(deployPath, 0755)
				destPath := filepath.Join(deployPath, filepath.Base(jarPath))
				if err := copyFile(jarPath, destPath); err != nil {
					updateOutput(fmt.Sprintf("复制JAR失败: %v\n", err))
				} else {
					updateOutput(fmt.Sprintf("部署JAR: %s -> %s\n", jarPath, destPath))
				}
			} else {
				updateOutput(fmt.Sprintf("未找到模块 %s 的JAR文件\n", module.Name))
			}

			if module.StartScript != "" {
				updateOutput(fmt.Sprintf("执行启动脚本: %s\n", module.StartScript))
				if err := runCommand(module.DeployDir, "bash", module.StartScript, "restart"); err != nil {
					log.Printf("启动脚本执行失败: %v", err)
				}
			}
		}
	}

	return nil
}

func buildFrontend(project *store.Project, config *store.Config, updateOutput func(string), force bool) error {
	updateOutput("开始构建前端项目...\n")

	env := os.Getenv("PATH")
	binDir := "bin"
	pathSep := ":"
	if runtime.GOOS == "windows" {
		binDir = "bin"
		pathSep = ";"
	}
	if config.NodeHome != "" {
		env = config.NodeHome + "/" + binDir + pathSep + env
	}

	updateOutput("执行 npm install...\n")
	if err := runCommandWithEnvAndOutput(project.LocalDir, "npm install", env, updateOutput); err != nil {
		updateOutput(fmt.Sprintf("npm install 失败: %v\n", err))
		return err
	}

	buildCmd := project.BuildCmd
	if buildCmd == "" {
		buildCmd = "npm run build"
	}

	updateOutput("执行构建命令: " + buildCmd + "\n")
	if err := runCommandWithEnvAndOutput(project.LocalDir, buildCmd, env, updateOutput); err != nil {
		updateOutput(fmt.Sprintf("构建失败: %v\n", err))
		return err
	}

	updateOutput("构建成功\n")

	if project.Modules != nil && len(project.Modules) > 0 && project.Modules[0].DeployDir != "" {
		deployDir := project.Modules[0].DeployDir
		os.MkdirAll(deployDir, 0755)
		distDir := filepath.Join(project.LocalDir, "dist")
		if _, err := os.Stat(distDir); err == nil {
			if err := copyDir(distDir, deployDir); err != nil {
				updateOutput(fmt.Sprintf("部署失败: %v\n", err))
			} else {
				updateOutput(fmt.Sprintf("部署成功: %s\n", deployDir))
			}
		}
	}

	return nil
}

func runCommand(dir string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandWithOutput(dir string, output func(string), name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandWithEnvAndOutput(dir string, cmdStr string, env string, output func(string)) error {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return fmt.Errorf("空命令")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "PATH="+env)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findLatestJar(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var latestFile os.FileInfo
	var latestPath string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".jar") {
			info, err := f.Info()
			if err == nil {
				if latestFile == nil || info.ModTime().After(latestFile.ModTime()) {
					latestFile = info
					latestPath = filepath.Join(dir, f.Name())
				}
			}
		}
	}
	return latestPath
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		return copyFile(path, dstPath)
	})
}

func SendWechatNotification(webhook string, content string) error {
	if webhook == "" {
		return nil
	}
	cmd := exec.Command("curl", "-s", webhook, "-H", "Content-Type: application/json",
		"-d", fmt.Sprintf(`{"msgtype": "text", "text": {"content": "%s"}}`, content))
	return cmd.Run()
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func getJavaVersion(javaHome string) string {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("where", "java")
	} else {
		cmd = exec.Command("which", "java")
	}
	if javaHome != "" {
		binDir := "bin"
		if runtime.GOOS == "windows" {
			binDir = "bin"
			cmd = exec.Command(filepath.Join(javaHome, binDir, "java"), "-version")
		} else {
			cmd = exec.Command(filepath.Join(javaHome, binDir, "java"), "-version")
		}
	} else {
		cmd = exec.Command("java", "-version")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "未找到"
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return string(out)
}

func getNodeVersion(nodeHome string) string {
	var cmd *exec.Cmd
	if nodeHome != "" {
		binDir := "bin"
		if runtime.GOOS == "windows" {
			cmd = exec.Command(filepath.Join(nodeHome, binDir, "node"), "-v")
		} else {
			cmd = exec.Command(filepath.Join(nodeHome, binDir, "node"), "-v")
		}
	} else {
		cmd = exec.Command("node", "-v")
	}
	out, err := cmd.Output()
	if err != nil {
		return "未找到"
	}
	return strings.TrimSpace(string(out))
}
