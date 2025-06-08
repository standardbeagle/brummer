package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// Simple test server on port 9999
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h1>Test Server</h1><p>Time: %s</p></body></html>", time.Now())
		})
		log.Println("Test server listening on :9999")
		http.ListenAndServe(":9999", nil)
	}()
	
	time.Sleep(1 * time.Second)
	
	// Test direct connection
	resp, err := http.Get("http://localhost:9999/")
	if err != nil {
		log.Fatal("Direct connection failed:", err)
	}
	resp.Body.Close()
	log.Println("Direct connection successful")
}