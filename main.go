package main

import (
	"dauto/service/router"
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
	flag.Parse()
	addr := fmt.Sprintf(":%d", *port)

	mux := http.NewServeMux()

	st, _ := fs.Sub(f, "webapp")
	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.FS(st))))

	router.SetupRoutesAPI(mux)

	fmt.Printf("Server starting on http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
