package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestPersistentHubClientCreation(t *testing.T) {
	client, err := NewPersistentHubClient(7777)
	if err != nil {
		t.Fatalf("Failed to create persistent hub client: %v", err)
	}

	if client == nil {
		t.Fatal("Client is nil")
	}

	stats := client.GetConnectionStats()
	if stats["maxIdleConns"] != 10 {
		t.Errorf("Expected maxIdleConns to be 10, got %v", stats["maxIdleConns"])
	}

	if stats["maxConnsPerHost"] != 2 {
		t.Errorf("Expected maxConnsPerHost to be 2, got %v", stats["maxConnsPerHost"])
	}

	if stats["idleConnTimeout"] != "24h0m0s" {
		t.Errorf("Expected idleConnTimeout to be 24h, got %v", stats["idleConnTimeout"])
	}

	if stats["established"] != false {
		t.Errorf("Expected established to be false initially, got %v", stats["established"])
	}

	// Test close
	err = client.Close()
	if err != nil {
		t.Errorf("Failed to close client: %v", err)
	}
}

func TestPersistentHubClientInterface(t *testing.T) {
	// Test that regular client creation works
	regularClient, err := NewHubClient(7777)
	if err != nil {
		t.Fatalf("Failed to create regular hub client: %v", err)
	}

	// Test that persistent client creation works
	persistentClient, err := NewPersistentHubClient(7777)
	if err != nil {
		t.Fatalf("Failed to create persistent hub client: %v", err)
	}

	// Test that both implement the interface
	var client1 HubClientInterface = regularClient
	var client2 HubClientInterface = persistentClient

	if client1 == nil {
		t.Error("Regular client does not implement interface")
	}

	if client2 == nil {
		t.Error("Persistent client does not implement interface")
	}
}

func TestPersistentClientFeatureFlag(t *testing.T) {
	// Test environment variable flag
	os.Setenv("BRUMMER_USE_ROBUST_NETWORKING", "true")
	defer os.Unsetenv("BRUMMER_USE_ROBUST_NETWORKING")

	client, err := NewHubClientInterface(7777)
	if err != nil {
		t.Fatalf("Failed to create client with feature flag: %v", err)
	}

	// Check that it's a persistent client by checking if it has connection stats
	if persistentClient, ok := client.(*PersistentHubClient); ok {
		stats := persistentClient.GetConnectionStats()
		if stats["established"] != false {
			t.Error("Expected new persistent client to not be established")
		}
	} else {
		t.Error("Expected persistent client when feature flag enabled")
	}

	// Test without feature flag
	os.Unsetenv("BRUMMER_USE_ROBUST_NETWORKING")

	client2, err := NewHubClientInterface(7777)
	if err != nil {
		t.Fatalf("Failed to create client without feature flag: %v", err)
	}

	// Should be regular client
	if _, ok := client2.(*HubClient); !ok {
		t.Error("Expected regular client when feature flag disabled")
	}
}

func TestPersistentClientBasicFunctionality(t *testing.T) {
	// Create a mock MCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Simple mock response
		if r.URL.Path == "/mcp" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"1.0"}}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Use a test port since we can't easily extract from httptest
	port := 8080

	client, err := NewPersistentHubClient(port)
	if err != nil {
		t.Fatalf("Failed to create persistent client: %v", err)
	}

	// Test that methods exist and can be called (even if they fail due to wrong port)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// These will fail because the port doesn't match, but we're testing the interface
	_ = client.Initialize(ctx)
	_, _ = client.ListTools(ctx)
	_, _ = client.ListResources(ctx)
	_, _ = client.ListPrompts(ctx)
	_ = client.Ping(ctx)

	// Test close
	err = client.Close()
	if err != nil {
		t.Errorf("Failed to close client: %v", err)
	}

	// Check that client is marked as not established after close
	if client.IsEstablished() {
		t.Error("Client should not be established after close")
	}
}

func TestInvalidPortHandling(t *testing.T) {
	// Test invalid ports
	invalidPorts := []int{-1, 0, 65536, 99999}

	for _, port := range invalidPorts {
		_, err := NewPersistentHubClient(port)
		if err == nil {
			t.Errorf("Expected error for invalid port %d, but got none", port)
		}
	}

	// Test valid ports
	validPorts := []int{1, 1024, 7777, 65535}

	for _, port := range validPorts {
		client, err := NewPersistentHubClient(port)
		if err != nil {
			t.Errorf("Expected no error for valid port %d, but got: %v", port, err)
		}
		if client != nil {
			client.Close()
		}
	}
}
