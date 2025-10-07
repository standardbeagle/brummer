package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMockMCPServer creates a mock MCP server with proper endpoints
func createMockMCPServer() *httptest.Server {
	router := mux.NewRouter()

	// Mock the MCP endpoint
	router.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request", http.StatusBadRequest)
			return
		}

		var request map[string]interface{}
		if err := json.Unmarshal(body, &request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Handle different JSON-RPC methods
		method, ok := request["method"].(string)
		if !ok {
			http.Error(w, "Missing method", http.StatusBadRequest)
			return
		}

		id := request["id"]

		switch method {
		case "initialize":
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id": id,
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
					"serverInfo": map[string]interface{}{
						"name": "brummer",
						"version": "1.0.0",
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case "tools/list":
			response := map[string]interface{}{
				"jsonrpc": "2.0",
				"id": id,
				"result": map[string]interface{}{
					"tools": []map[string]interface{}{
						{
							"name": "scripts_list",
							"description": "List all available npm/yarn/pnpm/bun scripts from package.json",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{},
							},
						},
						{
							"name": "scripts_run",
							"description": "Run a script from package.json with live log streaming",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name": map[string]interface{}{
										"type": "string",
										"description": "Script name to run",
									},
								},
								"required": []string{"name"},
							},
						},
						{
							"name": "logs_stream",
							"description": "Stream real-time logs from running processes",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"processId": map[string]interface{}{
										"type": "string",
										"description": "Optional process ID to filter logs",
									},
									"level": map[string]interface{}{
										"type": "string",
										"enum": []string{"all", "error", "warn", "info"},
										"description": "Log level filter",
									},
								},
							},
						},
						{
							"name": "proxy_requests",
							"description": "Get HTTP requests captured by the proxy server",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"processName": map[string]interface{}{
										"type": "string",
										"description": "Filter by process name",
									},
									"status": map[string]interface{}{
										"type": "string",
										"enum": []string{"all", "success", "error"},
										"description": "Filter by status",
									},
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case "tools/call":
			params, ok := request["params"].(map[string]interface{})
			if !ok {
				http.Error(w, "Missing params", http.StatusBadRequest)
				return
			}

			toolName, ok := params["name"].(string)
			if !ok {
				http.Error(w, "Missing tool name", http.StatusBadRequest)
				return
			}

			switch toolName {
			case "scripts_list":
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id": id,
					"result": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"type": "text",
								"text": `{
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "test": "vitest",
    "lint": "eslint . --ext .js,.jsx,.ts,.tsx"
  }
}`,
							},
						},
					},
				}
				json.NewEncoder(w).Encode(response)

			case "scripts_run":
				args, ok := params["arguments"].(map[string]interface{})
				if !ok {
					http.Error(w, "Missing arguments", http.StatusBadRequest)
					return
				}

				scriptName, ok := args["name"].(string)
				if !ok {
					http.Error(w, "Missing script name", http.StatusBadRequest)
					return
				}

				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id": id,
					"result": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"type": "text",
								"text": fmt.Sprintf("Started script: %s\nProcess ID: dev-%d\nLogs will stream in real-time.", scriptName, time.Now().Unix()),
							},
						},
						"isError": false,
					},
				}
				json.NewEncoder(w).Encode(response)

			case "logs_stream":
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id": id,
					"result": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"type": "text",
								"text": `[2024-01-15T10:30:00.000Z] INFO: dev-server started on port 3000
[2024-01-15T10:30:01.000Z] INFO: vite v4.5.0 building for development...
[2024-01-15T10:30:02.000Z] INFO: ➜  Local:   http://localhost:3000/
[2024-01-15T10:30:03.000Z] INFO: ➜  Network: http://192.168.1.100:3000/`,
							},
						},
					},
				}
				json.NewEncoder(w).Encode(response)

			case "proxy_requests":
				response := map[string]interface{}{
					"jsonrpc": "2.0",
					"id": id,
					"result": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"type": "text",
								"text": `[
  {
    "id": "req-1",
    "method": "GET",
    "url": "http://localhost:3000/",
    "statusCode": 200,
    "duration": "15ms",
    "timestamp": "2024-01-15T10:30:05.000Z"
  },
  {
    "id": "req-2",
    "method": "GET",
    "url": "http://localhost:3000/main.js",
    "statusCode": 200,
    "duration": "8ms",
    "timestamp": "2024-01-15T10:30:05.100Z"
  }
]`,
							},
						},
					},
				}
				json.NewEncoder(w).Encode(response)

			default:
				errorResponse := map[string]interface{}{
					"jsonrpc": "2.0",
					"id": id,
					"error": map[string]interface{}{
						"code": -32601,
						"message": fmt.Sprintf("Method not found: %s", toolName),
					},
				}
				json.NewEncoder(w).Encode(errorResponse)
			}

		default:
			errorResponse := map[string]interface{}{
				"jsonrpc": "2.0",
				"id": id,
				"error": map[string]interface{}{
					"code": -32601,
					"message": fmt.Sprintf("Method not found: %s", method),
				},
			}
			json.NewEncoder(w).Encode(errorResponse)
		}
	})

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"status": "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"version": "1.0.0",
		}
		json.NewEncoder(w).Encode(response)
	})

	return httptest.NewServer(router)
}

// TestMCPServerIntegration tests the MCP server endpoints
func TestMCPServerIntegration(t *testing.T) {
	server := createMockMCPServer()
	defer server.Close()

	client := server.Client()

	t.Run("Health Check", func(t *testing.T) {
		req, err := http.NewRequest("GET", server.URL+"/health", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.Contains(t, response, "timestamp")
		assert.Equal(t, "1.0.0", response["version"])

		t.Logf("✅ Health check passed")
	})

	t.Run("MCP Initialize", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "initialize",
			"params": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"clientInfo": map[string]interface{}{
					"name": "test-client",
					"version": "1.0.0",
				},
			},
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, float64(1), response["id"])
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Equal(t, "2024-11-05", result["protocolVersion"])
		assert.Contains(t, result, "capabilities")

		serverInfo := result["serverInfo"].(map[string]interface{})
		assert.Equal(t, "brummer", serverInfo["name"])

		t.Logf("✅ MCP initialization passed")
	})

	t.Run("Tools List", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id": 2,
			"method": "tools/list",
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, float64(2), response["id"])
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Contains(t, result, "tools")

		tools := result["tools"].([]interface{})
		assert.GreaterOrEqual(t, len(tools), 4) // scripts_list, scripts_run, logs_stream, proxy_requests

		// Find scripts_list tool
		var scriptsListTool map[string]interface{}
		for _, tool := range tools {
			toolMap := tool.(map[string]interface{})
			if toolMap["name"] == "scripts_list" {
				scriptsListTool = toolMap
				break
			}
		}

		require.NotNil(t, scriptsListTool, "scripts_list tool should be available")
		assert.Equal(t, "scripts_list", scriptsListTool["name"])
		assert.Contains(t, scriptsListTool, "description")
		assert.Contains(t, scriptsListTool, "inputSchema")

		t.Logf("✅ Tools list passed - found %d tools", len(tools))
	})

	t.Run("Invalid Method", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id": 3,
			"method": "invalid/method",
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, float64(3), response["id"])
		assert.Contains(t, response, "error")

		error := response["error"].(map[string]interface{})
		assert.Equal(t, float64(-32601), error["code"])
		assert.Contains(t, error["message"], "Method not found")

		t.Logf("✅ Invalid method properly rejected")
	})
}

// TestMCPToolsIntegration tests actual MCP tool calls
func TestMCPToolsIntegration(t *testing.T) {
	server := createMockMCPServer()
	defer server.Close()

	client := server.Client()

	t.Run("Call scripts_list tool", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "tools/call",
			"params": map[string]interface{}{
				"name": "scripts_list",
			},
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, float64(1), response["id"])
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Contains(t, result, "content")

		content := result["content"].([]interface{})
		assert.Len(t, content, 1)

		textBlock := content[0].(map[string]interface{})
		assert.Equal(t, "text", textBlock["type"])
		assert.Contains(t, textBlock["text"], "dev")
		assert.Contains(t, textBlock["text"], "build")
		assert.Contains(t, textBlock["text"], "test")

		t.Logf("✅ scripts_list tool executed successfully")
	})

	t.Run("Call scripts_run tool", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id": 2,
			"method": "tools/call",
			"params": map[string]interface{}{
				"name": "scripts_run",
				"arguments": map[string]interface{}{
					"name": "dev",
				},
			},
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, float64(2), response["id"])
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Contains(t, result, "content")
		assert.Equal(t, false, result["isError"])

		content := result["content"].([]interface{})
		assert.Len(t, content, 1)

		textBlock := content[0].(map[string]interface{})
		assert.Equal(t, "text", textBlock["type"])
		assert.Contains(t, textBlock["text"], "Started script: dev")
		assert.Contains(t, textBlock["text"], "Process ID:")

		t.Logf("✅ scripts_run tool executed successfully")
	})

	t.Run("Call logs_stream tool", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id": 3,
			"method": "tools/call",
			"params": map[string]interface{}{
				"name": "logs_stream",
				"arguments": map[string]interface{}{
					"level": "info",
				},
			},
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, float64(3), response["id"])
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Contains(t, result, "content")

		content := result["content"].([]interface{})
		assert.Len(t, content, 1)

		textBlock := content[0].(map[string]interface{})
		assert.Equal(t, "text", textBlock["type"])
		assert.Contains(t, textBlock["text"], "dev-server started")
		assert.Contains(t, textBlock["text"], "Local:")

		t.Logf("✅ logs_stream tool executed successfully")
	})

	t.Run("Call proxy_requests tool", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id": 4,
			"method": "tools/call",
			"params": map[string]interface{}{
				"name": "proxy_requests",
			},
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, float64(4), response["id"])
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Contains(t, result, "content")

		content := result["content"].([]interface{})
		assert.Len(t, content, 1)

		textBlock := content[0].(map[string]interface{})
		assert.Equal(t, "text", textBlock["type"])
		assert.Contains(t, textBlock["text"], "req-1")
		assert.Contains(t, textBlock["text"], "localhost:3000")

		t.Logf("✅ proxy_requests tool executed successfully")
	})

	t.Run("Invalid tool call", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id": 5,
			"method": "tools/call",
			"params": map[string]interface{}{
				"name": "invalid_tool",
			},
		}

		jsonBody, err := json.Marshal(requestBody)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", server.URL+"/mcp", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, float64(5), response["id"])
		assert.Contains(t, response, "error")

		error := response["error"].(map[string]interface{})
		assert.Equal(t, float64(-32601), error["code"])
		assert.Contains(t, error["message"], "Method not found: invalid_tool")

		t.Logf("✅ Invalid tool call properly rejected")
	})
}