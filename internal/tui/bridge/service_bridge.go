package bridge

import (
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

// ServiceBridge is the adapter layer that converts service events into BubbleTea messages.
// This is the ONLY place where services communicate with the UI.
//
// Design principles:
// 1. Services never know about components
// 2. Components never call service methods directly
// 3. All service state changes flow through this bridge as messages
// 4. Bridge subscribes to service events and converts them to tea.Msg
// 5. BubbleTea program.Send() is thread-safe and non-blocking
type ServiceBridge struct {
	processMgr  *process.Manager
	logStore    *logs.Store
	proxyServer *proxy.Server
	eventBus    *events.EventBus
	program     Program

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
	return &ServiceBridge{
		processMgr:    processMgr,
		logStore:      logStore,
		proxyServer:   proxyServer,
		eventBus:      eventBus,
		subscriptions: []func(){},
	}
}

// Start begins converting service events to BubbleTea messages.
// The program parameter is used to send messages to the BubbleTea event loop.
//
// This must be called AFTER the tea.Program is created but BEFORE program.Run().
func (b *ServiceBridge) Start(program Program) {
	b.program = program

	// Subscribe to all service event sources
	b.subscribeToProcessManager()
	b.subscribeToLogStore()
	b.subscribeToEventBus()
	b.subscribeToProxyServer()
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
		if b.program != nil {
			b.program.Send(messages.LogAddedMsg{
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
		if b.program != nil {
			name := ""
			script := ""
			if n, ok := evt.Data["name"].(string); ok {
				name = n
			}
			if s, ok := evt.Data["script"].(string); ok {
				script = s
			}

			b.program.Send(messages.ProcessStartedMsg{
				ProcessID: evt.ProcessID,
				Name:      name,
				Script:    script,
				StartTime: time.Now(),
			})
		}
	})

	// Process exited events (map to ProcessStoppedMsg)
	b.eventBus.Subscribe(events.ProcessExited, func(evt events.Event) {
		if b.program != nil {
			name := ""
			exitCode := -1
			if n, ok := evt.Data["name"].(string); ok {
				name = n
			}
			if ec, ok := evt.Data["exit_code"].(int); ok {
				exitCode = ec
			}

			b.program.Send(messages.ProcessStoppedMsg{
				ProcessID: evt.ProcessID,
				Name:      name,
				ExitCode:  exitCode,
				EndTime:   time.Now(),
			})
		}
	})

	// URL detection events
	b.eventBus.Subscribe(events.URLDetected, func(evt events.Event) {
		if b.program != nil {
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

			b.program.Send(messages.URLDetectedMsg{
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
	// Future: Add proxy request/response events here if needed
}

// Stop cleanly shuts down the ServiceBridge
func (b *ServiceBridge) Stop() {
	// Unsubscribe from LogStore observers
	for _, unsub := range b.subscriptions {
		if unsub != nil {
			unsub()
		}
	}
	b.subscriptions = nil
	b.program = nil

	// Note: EventBus subscriptions don't have unsubscribe in this version
	// If needed, can be implemented by tracking handlers and removing them
}