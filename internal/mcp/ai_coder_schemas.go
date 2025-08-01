package mcp

import "encoding/json"

// AI Coder tool schemas for parameter validation

var aiCoderCreateSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"task": {
			"type": "string",
			"description": "The coding task or request for the AI coder to perform",
			"minLength": 1,
			"maxLength": 2000
		},
		"provider": {
			"type": "string",
			"description": "AI provider to use (claude, gpt4, local, mock)",
			"enum": ["claude", "gpt4", "local", "mock"],
			"default": "mock"
		},
		"workspace_files": {
			"type": "array",
			"description": "Initial files to include in AI coder workspace context",
			"items": {
				"type": "string"
			},
			"maxItems": 50
		},
		"name": {
			"type": "string",
			"description": "Optional friendly name for the AI coder session",
			"maxLength": 100
		}
	},
	"required": ["task"],
	"additionalProperties": false
}`)

var aiCoderListSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"status_filter": {
			"type": "string",
			"description": "Filter coders by status (running, completed, failed, paused, all)",
			"enum": ["running", "completed", "failed", "paused", "stopped", "creating", "all"],
			"default": "all"
		},
		"limit": {
			"type": "integer",
			"description": "Maximum number of coders to return",
			"minimum": 1,
			"maximum": 100,
			"default": 20
		}
	},
	"additionalProperties": false
}`)

var aiCoderControlSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"coder_id": {
			"type": "string",
			"description": "The ID of the AI coder to control"
		},
		"action": {
			"type": "string",
			"description": "Control action to perform",
			"enum": ["start", "pause", "resume", "stop"]
		}
	},
	"required": ["coder_id", "action"],
	"additionalProperties": false
}`)

var aiCoderStatusSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"coder_id": {
			"type": "string",
			"description": "The ID of the AI coder to get status for"
		}
	},
	"required": ["coder_id"],
	"additionalProperties": false
}`)

var aiCoderWorkspaceSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"coder_id": {
			"type": "string",
			"description": "The ID of the AI coder whose workspace to access"
		},
		"operation": {
			"type": "string",
			"description": "Operation to perform on workspace",
			"enum": ["list", "read"],
			"default": "list"
		},
		"file_path": {
			"type": "string",
			"description": "File path to read (required for read operation)"
		}
	},
	"required": ["coder_id"],
	"additionalProperties": false
}`)

var aiCoderLogsSchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"coder_id": {
			"type": "string",
			"description": "The ID of the AI coder to get logs from"
		},
		"limit": {
			"type": "integer",
			"description": "Maximum number of log entries to return",
			"minimum": 1,
			"maximum": 1000,
			"default": 100
		},
		"follow": {
			"type": "boolean",
			"description": "Stream logs in real-time (only for streaming handler)",
			"default": true
		},
		"output_file": {
			"type": "string",
			"description": "Optional file path to write log data (e.g., 'ai-logs.json', 'debug/coder-logs.json')"
		}
	},
	"required": ["coder_id"],
	"additionalProperties": false
}`)
