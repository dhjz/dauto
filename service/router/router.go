package router

import "net/http"

func SetupRoutesAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello Vue!"))
	})
}
