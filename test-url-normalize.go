package main

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	
	// Remove trailing slashes from path
	path := strings.TrimSuffix(u.Path, "/")
	if path == "" {
		path = "/"
	}
	
	// Rebuild URL without query params for mapping
	normalized := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path)
	return normalized
}

func main() {
	tests := []string{
		"http://localhost:3000",
		"http://localhost:3000/",
		"http://localhost:3000/api",
		"http://localhost:3000/api/",
		"http://localhost:3000/api?foo=bar",
	}
	
	for _, test := range tests {
		fmt.Printf("%-40s -> %s\n", test, normalizeURL(test))
	}
}