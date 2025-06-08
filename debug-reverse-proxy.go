package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func main() {
	fmt.Println("Testing reverse proxy on port 20888...")
	
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get("http://localhost:20888/")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading body: %v\n", err)
		return
	}
	
	fmt.Printf("Body length: %d\n", len(body))
	if len(body) < 500 {
		fmt.Printf("Body: %s\n", string(body))
	} else {
		fmt.Printf("Body (first 500 chars): %s...\n", string(body[:500]))
	}
}