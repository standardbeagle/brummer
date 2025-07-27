package mcp

import (
	"log"
	"sync/atomic"
)

// debugEnabled controls whether MCP debug logging is enabled
var debugEnabled int32

// SetDebugEnabled sets whether MCP debug logging is enabled
func SetDebugEnabled(enabled bool) {
	if enabled {
		atomic.StoreInt32(&debugEnabled, 1)
	} else {
		atomic.StoreInt32(&debugEnabled, 0)
	}
}

// IsDebugEnabled returns whether MCP debug logging is enabled
func IsDebugEnabled() bool {
	return atomic.LoadInt32(&debugEnabled) == 1
}

// debugLog logs a message only if MCP debug logging is enabled
func debugLog(format string, args ...interface{}) {
	if IsDebugEnabled() {
		log.Printf("[MCP] "+format, args...)
	}
}
