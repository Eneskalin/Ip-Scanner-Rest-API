package main

import (
	"IpScanner/config"
	"IpScanner/handlers"
	"fmt"
	"net/http"
)

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {

	mux := http.NewServeMux()
	if config.AppConfig.WithGui {
		fs := http.FileServer(http.Dir("static"))
		mux.Handle("/", fs)
	}

	mux.HandleFunc("/scanip", handlers.ScanIp)
	mux.HandleFunc("/ping", handlers.Ping)

	handler := enableCORS(mux)

	fmt.Println("Server starting on port 8080...")
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}
