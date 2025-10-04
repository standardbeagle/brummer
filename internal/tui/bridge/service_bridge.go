package bridge

import (
	"fmt"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	"github.com/standardbeagle/brummer/internal/process"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/pkg/events"
)

// Program interface abstracts tea.Program for testing
type Program interface {
	Send(tea.Msg)
}

// programHolder wraps Program to allow atomic.Value to store nil
type programHolder struct {
	prog Program
}

// ServiceBridge is the adapter layer that converts service events into BubbleTea messages.
// This is the ONLY place where services communicate with the UI.
//
// Design principles:
// 1. Services never know about components
// 2. Components never call service methods directly
// 3. All service state changes flow through this bridge as messages
// 4. Bridge subscribes to service events and converts them to tea.Msg
// 5. BubbleTea program.Send() is thread-safe and non-blocking
// 6. atomic.Value provides lock-free thread-safe access to program reference
type ServiceBridge struct {
	processMgr  *process.Manager
	logStore    *logs.Store
	proxyServer *proxy.Server
	eventBus    *events.EventBus

	// Thread-safe program reference using atomic.Value
	// Stores *programHolder (never nil, but holder.prog can be nil)
	program atomic.Value // *programHolder

	// Cleanup functions for subscriptions
	subscriptions []func()
}

// NewServiceBridge creates a new service-to-UI adapter.
// The program reference will be set when Start() is called.
func NewServiceBridge(
	processMgr *process.Manager,
	logStore *logs.Store,
	proxyServer *proxy.Server,
	eventBus *events.EventBus,
) *ServiceBridge {
	sb := &ServiceBridge{
		processMgr:    processMgr,
		logStore:      logStore,
		proxyServer:   proxyServer,
		eventBus:      eventBus,
		subscriptions: []func(){},
	}
	// Initialize atomic.Value with empty holder
	sb.program.Store(&programHolder{prog: nil})
	return sb
}

// Start begins converting service events to BubbleTea messages.
// The program parameter is used to send messages to the BubbleTea event loop.
//
// This must be called AFTER the tea.Program is created but BEFORE program.Run().
func (b *ServiceBridge) Start(program Program) {
	// Store program reference atomically (lock-free, thread-safe)
	b.program.Store(&programHolder{prog: program})

	// Subscribe to all service event sources
	b.subscribeToProcessManager()
	b.subscribeToLogStore()
	b.subscribeToEventBus()
	b.subscribeToProxyServer()

	// Send initial status updates for services that are already running
	b.sendInitialStatusUpdates()
}

// getProgram atomically loads the program reference (lock-free, thread-safe)
func (b *ServiceBridge) getProgram() Program {
	holder := b.program.Load().(*programHolder)
	return holder.prog
}

// subscribeToProcessManager converts ProcessManager callbacks to messages
func (b *ServiceBridge) subscribeToProcessManager() {
	if b.processMgr == nil {
		return
	}

	// Convert process log output to messages
	// NOTE: These logs will be added to LogStore, which will emit LogAddedMsg
	// So we don't need to send ProcessOutputMsg directly - it's handled by LogStore observer
	b.processMgr.AddLogCallback(func(processID, line string, isError bool) {
		// This callback adds to LogStore, which triggers LogAddedMsg via observer
		// We include this for completeness, but the real message flow is:
		// ProcessManager.EmitLog() → LogStore.Add() → Observer → LogAddedMsg
	})
}

// subscribeToLogStore converts LogStore observer events to messages
func (b *ServiceBridge) subscribeToLogStore() {
	if b.logStore == nil {
		return
	}

	// Subscribe to log additions using observer pattern
	unsubscribe := b.logStore.OnAdd(func(entry *logs.LogEntry) {
		// Atomically load program reference (lock-free, thread-safe)
		if prog := b.getProgram(); prog != nil {
			prog.Send(messages.LogAddedMsg{
				Entry: entry,
			})
		}
	})

	b.subscriptions = append(b.subscriptions, unsubscribe)
}

// subscribeToEventBus converts EventBus events to messages
func (b *ServiceBridge) subscribeToEventBus() {
	if b.eventBus == nil {
		return
	}

	// Process started events
	b.eventBus.Subscribe(events.ProcessStarted, func(evt events.Event) {
		// Atomically load program reference (lock-free, thread-safe)
		if prog := b.getProgram(); prog != nil {
			name := ""
			script := ""
			if n, ok := evt.Data["name"].(string); ok {
				name = n
			}
			if s, ok := evt.Data["script"].(string); ok {
				script = s
			}

			prog.Send(messages.ProcessStartedMsg{
				ProcessID: evt.ProcessID,
				Name:      name,
				Script:    script,
				StartTime: time.Now(),
			})
		}
	})

	// Process started events (also update process count)
	b.eventBus.Subscribe(events.ProcessStarted, func(evt events.Event) {
		// Atomically load program reference (lock-free, thread-safe)
		if prog := b.getProgram(); prog != nil {
			name := ""
			script := ""
			if n, ok := evt.Data["name"].(string); ok {
				name = n
			}
			if s, ok := evt.Data["script"].(string); ok {
				script = s
			}

			prog.Send(messages.ProcessStartedMsg{
				ProcessID: evt.ProcessID,
				Name:      name,
				Script:    script,
				StartTime: time.Now(),
			})

			// Also send process count update
			if b.processMgr != nil {
				if processes := b.processMgr.GetAllProcesses(); len(processes) > 0 {
					prog.Send(messages.ServiceStatusUpdateMsg{
						Service: "processes",
						Status:  "running",
						Details: fmt.Sprintf("%d active", len(processes)),
					})
				}
			}
		}
	})

	// Process exited events (map to ProcessStoppedMsg)
	b.eventBus.Subscribe(events.ProcessExited, func(evt events.Event) {
		// Atomically load program reference (lock-free, thread-safe)
		if prog := b.getProgram(); prog != nil {
			name := ""
			exitCode := -1
			if n, ok := evt.Data["name"].(string); ok {
				name = n
			}
			if ec, ok := evt.Data["exit_code"].(int); ok {
				exitCode = ec
			}

			prog.Send(messages.ProcessStoppedMsg{
				ProcessID: evt.ProcessID,
				Name:      name,
				ExitCode:  exitCode,
				EndTime:   time.Now(),
			})

			// Also send process count update
			if b.processMgr != nil {
				if processes := b.processMgr.GetAllProcesses(); len(processes) > 0 {
					prog.Send(messages.ServiceStatusUpdateMsg{
						Service: "processes",
						Status:  "running",
						Details: fmt.Sprintf("%d active", len(processes)),
					})
				} else {
					prog.Send(messages.ServiceStatusUpdateMsg{
						Service: "processes",
						Status:  "stopped",
						Details: "No active processes",
					})
				}
			}
		}
	})

	// MCP activity events
	b.eventBus.Subscribe(events.MCPConnected, func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			prog.Send(messages.ServiceStatusUpdateMsg{
				Service: "mcp",
				Status:  "running",
				Details: "MCP server connected",
			})
		}
	})

	b.eventBus.Subscribe(events.MCPDisconnected, func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			prog.Send(messages.ServiceStatusUpdateMsg{
				Service: "mcp",
				Status:  "stopped",
				Details: "MCP server disconnected",
			})
		}
	})

	// AI coder events
	b.eventBus.Subscribe(events.EventAICoderStarted, func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			details := "AI coder started"
			if name, ok := evt.Data["coder_name"].(string); ok {
				details = fmt.Sprintf("%s started", name)
			}
			prog.Send(messages.ServiceStatusUpdateMsg{
				Service: "ai_coders",
				Status:  "running",
				Details: details,
			})
		}
	})

	b.eventBus.Subscribe(events.EventAICoderStopped, func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			details := "AI coder stopped"
			if name, ok := evt.Data["coder_name"].(string); ok {
				details = fmt.Sprintf("%s stopped", name)
			}
			prog.Send(messages.ServiceStatusUpdateMsg{
				Service: "ai_coders",
				Status:  "stopped",
				Details: details,
			})
		}
	})

	b.eventBus.Subscribe(events.EventAICoderFailed, func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			details := "AI coder failed"
			if name, ok := evt.Data["coder_name"].(string); ok {
				details = fmt.Sprintf("%s failed", name)
			}
			if errMsg, ok := evt.Data["error_message"].(string); ok && errMsg != "" {
				details += ": " + errMsg
			}
			prog.Send(messages.ServiceStatusUpdateMsg{
				Service: "ai_coders",
				Status:  "error",
				Details: details,
			})
		}
	})

	// Proxy request events for web view
	b.eventBus.Subscribe(events.ProxyRequest, func(evt events.Event) {
		// Atomically load program reference (lock-free, thread-safe)
		if prog := b.getProgram(); prog != nil {
			method := ""
			url := ""
			statusCode := 0
			durationMs := 0
			processName := ""

			if m, ok := evt.Data["method"].(string); ok {
				method = m
			}
			if u, ok := evt.Data["url"].(string); ok {
				url = u
			}
			if sc, ok := evt.Data["status"].(int); ok {
				statusCode = sc
			}
			if dm, ok := evt.Data["duration"].(int); ok {
				durationMs = dm
			}
			if pn, ok := evt.Data["processName"].(string); ok {
				processName = pn
			}

			// Convert milliseconds to time.Duration
			duration := time.Duration(durationMs) * time.Millisecond

			prog.Send(messages.ProxyRequestMsg{
				RequestID:   fmt.Sprintf("%d", time.Now().UnixNano()), // Generate unique ID
				Method:      method,
				URL:         url,
				Path:        "", // Extract from URL if needed
				StatusCode:  statusCode,
				Duration:    duration,
				ProcessName: processName,
				Timestamp:   time.Now(),
			})
		}
	})

	// URL detection events
	b.eventBus.Subscribe(events.URLDetected, func(evt events.Event) {
		// Atomically load program reference (lock-free, thread-safe)
		if prog := b.getProgram(); prog != nil {
			url := ""
			urlType := "http"
			hasProxy := false
			proxyPort := 0
			proxyURL := ""
			proxyLabel := ""

			if u, ok := evt.Data["url"].(string); ok {
				url = u
			}
			if t, ok := evt.Data["type"].(string); ok {
				urlType = t
			}
			if p, ok := evt.Data["has_proxy"].(bool); ok {
				hasProxy = p
			}
			if pp, ok := evt.Data["proxy_port"].(int); ok {
				proxyPort = pp
			}
			if pu, ok := evt.Data["proxy_url"].(string); ok {
				proxyURL = pu
			}
			if pl, ok := evt.Data["proxy_label"].(string); ok {
				proxyLabel = pl
			}

			prog.Send(messages.URLDetectedMsg{
				URL:        url,
				ProcessID:  evt.ProcessID,
				Type:       urlType,
				Timestamp:  time.Now(),
				HasProxy:   hasProxy,
				ProxyPort:  proxyPort,
				ProxyURL:   proxyURL,
				ProxyLabel: proxyLabel,
			})
		}
	})

	// Note: EventBus.Subscribe() doesn't return unsubscribe functions in this version
	// So we don't track subscriptions for cleanup. If needed, can be added in future.
}

// subscribeToProxyServer converts proxy server events to messages
func (b *ServiceBridge) subscribeToProxyServer() {
	if b.proxyServer == nil {
		return
	}

	// Proxy server doesn't currently emit events beyond URL detection (handled by EventBus)
	// System message events (proxy registration, errors, etc.)
	b.eventBus.Subscribe(events.SystemMessage, func(evt events.Event) {
		// Atomically load program reference (lock-free, thread-safe)
		if prog := b.getProgram(); prog != nil {
			level := "info"
			context := ""
			message := ""

			if l, ok := evt.Data["level"].(string); ok {
				level = l
			}
			if c, ok := evt.Data["context"].(string); ok {
				context = c
			}
			if m, ok := evt.Data["message"].(string); ok {
				message = m
			}

			// Create a formatted log entry
			formattedMsg := fmt.Sprintf("[%s] %s: %s", context, message, level)

			// Add to appropriate log store based on level
			if level == "error" {
				b.logStore.Add("system", context, formattedMsg, true)
			} else {
				b.logStore.Add("system", context, formattedMsg, false)
			}
		}
	})

	// Future: Add proxy request/response events here if needed
}

// Stop cleanly shuts down the ServiceBridge
func (b *ServiceBridge) Stop() {
	// Atomically clear program reference (lock-free, thread-safe)
	// This prevents observers from sending messages after shutdown
	b.program.Store(&programHolder{prog: nil})

	// Unsubscribe from LogStore observers
	for _, unsub := range b.subscriptions {
		if unsub != nil {
			unsub()
		}
	}
	b.subscriptions = nil

	// Note: EventBus subscriptions don't have unsubscribe in this version
	// If needed, can be implemented by tracking handlers and removing them
}

// sendInitialStatusUpdates sends status updates for services that are already running
func (b *ServiceBridge) sendInitialStatusUpdates() {
	prog := b.getProgram()
	if prog == nil {
		return
	}

	// Check process manager for existing processes
	if b.processMgr != nil {
		if processes := b.processMgr.GetAllProcesses(); len(processes) > 0 {
			prog.Send(messages.ServiceStatusUpdateMsg{
				Service: "processes",
				Status:  "running",
				Details: fmt.Sprintf("%d active", len(processes)),
			})
		} else {
			prog.Send(messages.ServiceStatusUpdateMsg{
				Service: "processes",
				Status:  "stopped",
				Details: "No active processes",
			})
		}
	}

	// Check proxy server status
	if b.proxyServer != nil && b.proxyServer.IsRunning() {
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: "proxy",
			Status:  "running",
			Details: fmt.Sprintf("Port %d", b.proxyServer.GetPort()),
		})
	} else {
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: "proxy",
			Status:  "stopped",
			Details: "Proxy server stopped",
		})
	}
}
