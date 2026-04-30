package main

import (
	"dauto/service/executor"
	"dauto/service/router"
	"dauto/service/scheduler"
	"dauto/service/store"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
)

//go:embed all:webapp
var f embed.FS

func main() {
	port := flag.Int("p", 8002, "server port")
	password := flag.String("pwd", "111", "password")
	flag.Parse()

	if err := store.Init(); err != nil {
		log.Printf("初始化存储失败: %v", err)
	}
	log.Printf("登录密码: " + *password)
	s := store.GetStore()
	javaHome := s.Config.JavaHome
	mavenHome := s.Config.MavenHome
	nodeHome := s.Config.NodeHome
	fmt.Printf("Java: %s\n", executor.GetJavaVersion(javaHome))
	fmt.Printf("Maven: %s\n", executor.GetMavenVersion(mavenHome))
	fmt.Printf("Node: %s\n", executor.GetNodeVersion(nodeHome))

	scheduler.Start()

	addr := fmt.Sprintf(":%d", *port)

	mux := http.NewServeMux()

	st, _ := fs.Sub(f, "webapp")
	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.FS(st))))

	router.SetupRoutesAPI(mux, *password)

	fmt.Printf("自动化部署服务启动于 http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
