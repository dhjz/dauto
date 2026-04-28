package executor

import (
	"dauto/service/store"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunProject(execID string, projectID string) (string, error) {
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

	if err := os.MkdirAll(project.LocalDir, 0755); err != nil {
		updateOutput(fmt.Sprintf("创建目录失败: %v\n", err))
		return "", fmt.Errorf("创建目录失败: %v", err)
	}

	if _, err := os.Stat(filepath.Join(project.LocalDir, ".git")); os.IsNotExist(err) {
		updateOutput("克隆仓库...\n")
		if err := runCommandWithOutput("", updateOutput, "git", "clone", "-b", project.Branch, project.RepoURL, project.LocalDir); err != nil {
			updateOutput(fmt.Sprintf("克隆仓库失败: %v\n", err))
			return "", fmt.Errorf("克隆仓库失败: %v", err)
		}
	} else {
		updateOutput("更新仓库...\n")
		if err := runCommandWithOutput(project.LocalDir, updateOutput, "git", "fetch", "origin", project.Branch); err != nil {
			log.Printf("git fetch 警告: %v", err)
		}
		if err := runCommandWithOutput(project.LocalDir, updateOutput, "git", "checkout", project.Branch); err != nil {
			log.Printf("git checkout 警告: %v", err)
		}
		if err := runCommandWithOutput(project.LocalDir, updateOutput, "git", "pull", "origin", project.Branch); err != nil {
			log.Printf("git pull 警告: %v", err)
		}
	}

	if project.Type == "backend" {
		err := buildBackend(project, config, updateOutput)
		return getFinalOutput(execID), err
	} else if project.Type == "frontend" {
		err := buildFrontend(project, config, updateOutput)
		return getFinalOutput(execID), err
	}

	return getFinalOutput(execID), nil
}

func getFinalOutput(execID string) string {
	exec := store.GetExecution(execID)
	if exec != nil {
		return exec.Output
	}
	return ""
}

func buildBackend(project *store.Project, config *store.Config, updateOutput func(string)) error {
	updateOutput("开始构建后端项目...\n")

	env := os.Getenv("PATH")
	if config.MavenHome != "" {
		env = config.MavenHome + "/bin:" + env
	}
	if config.JavaHome != "" {
		env = config.JavaHome + "/bin:" + env
	}

	buildCmd := project.BuildCmd
	if buildCmd == "" {
		buildCmd = "mvn clean package -DskipTests"
	}

	if err := runCommandWithEnvAndOutput(project.LocalDir, buildCmd, env, updateOutput); err != nil {
		updateOutput(fmt.Sprintf("构建失败: %v\n", err))
		return err
	}

	updateOutput("构建成功\n")

	if project.DeployDir != "" {
		os.MkdirAll(project.DeployDir, 0755)

		for _, module := range project.Modules {
			jarPath := findJarFile(project.LocalDir + "/" + module + "/target")
			if jarPath != "" {
				deployPath := filepath.Join(project.DeployDir, module)
				os.MkdirAll(deployPath, 0755)
				destPath := filepath.Join(deployPath, filepath.Base(jarPath))
				if err := copyFile(jarPath, destPath); err != nil {
					updateOutput(fmt.Sprintf("复制JAR失败: %v\n", err))
				} else {
					updateOutput(fmt.Sprintf("部署JAR: %s -> %s\n", jarPath, destPath))
				}
			}
		}

		if project.StartScript != "" {
			updateOutput(fmt.Sprintf("执行启动脚本: %s\n", project.StartScript))
			if err := runCommand(project.DeployDir, "bash", project.StartScript, "restart"); err != nil {
				log.Printf("启动脚本执行失败: %v", err)
			}
		}
	}

	return nil
}

func buildFrontend(project *store.Project, config *store.Config, updateOutput func(string)) error {
	updateOutput("开始构建前端项目...\n")

	env := os.Getenv("PATH")
	if config.NodeHome != "" {
		env = config.NodeHome + "/bin:" + env
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

	updateOutput("执行构建命令...\n")
	if err := runCommandWithEnvAndOutput(project.LocalDir, buildCmd, env, updateOutput); err != nil {
		updateOutput(fmt.Sprintf("构建失败: %v\n", err))
		return err
	}

	updateOutput("构建成功\n")

	if project.DeployDir != "" && project.Type == "frontend" {
		os.MkdirAll(project.DeployDir, 0755)
		distDir := filepath.Join(project.LocalDir, "dist")
		if _, err := os.Stat(distDir); err == nil {
			if err := copyDir(distDir, project.DeployDir); err != nil {
				updateOutput(fmt.Sprintf("部署失败: %v\n", err))
			} else {
				updateOutput(fmt.Sprintf("部署成功: %s\n", project.DeployDir))
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

func findJarFile(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".jar") {
			return filepath.Join(dir, f.Name())
		}
	}
	return ""
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
