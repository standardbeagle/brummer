package bridge

import (
	"fmt"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/logs"
	
	processes "github.com/standardbeagle/brummer/internal/domain/processes"
	"github.com/standardbeagle/brummer/internal/proxy"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
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

// TestManagementService interface defines the contract for test management services
type TestManagementService interface {
	GetServiceName() string
	GetServiceStatus() services.ServiceStatus
	SubscribeToEvents()
	Stop() error
}

// FileBrowserService interface defines the contract for file browser services
type FileBrowserService interface {
	GetServiceName() string
	GetServiceStatus() services.ServiceStatus
	SubscribeToEvents()
	Stop() error
}

// SettingsService interface defines the contract for settings services
type SettingsService interface {
	GetServiceName() string
	GetServiceStatus() services.ServiceStatus
	SubscribeToEvents()
	Stop() error
}

// InputService interface defines the contract for input services
type InputService interface {
	GetServiceName() string
	GetServiceStatus() services.ServiceStatus
	SubscribeToEvents()
	Stop() error
}

// NavigationService interface defines the contract for navigation services
type NavigationService interface {
	GetServiceName() string
	GetServiceStatus() services.ServiceStatus
	SubscribeToEvents()
	Stop() error
}

// SystemService interface defines the contract for system services
type SystemService interface {
	GetServiceName() string
	GetServiceStatus() services.ServiceStatus
	SubscribeToEvents()
	Stop() error
}

// NotificationsService interface defines the contract for notifications services
type NotificationsService interface {
	GetServiceName() string
	GetServiceStatus() services.ServiceStatus
	SubscribeToEvents()
	Stop() error
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
	processMgr            *processes.Manager
	logStore              *logs.Store
	proxyServer           *proxy.Server
	eventBus              *events.EventBus
	testManagementService TestManagementService // Interface for test management
	fileBrowserService    FileBrowserService    // Interface for file browser
	settingsService       SettingsService       // Interface for settings
	inputService          InputService          // Interface for input
	navigationService     NavigationService     // Interface for navigation
	systemService         SystemService         // Interface for system
	notificationsService  NotificationsService  // Interface for notifications

	// Thread-safe program reference using atomic.Value
	// Stores *programHolder (never nil, but holder.prog can be nil)
	program atomic.Value // *programHolder

	// Cleanup functions for subscriptions
	subscriptions []func()
}

// NewServiceBridge creates a new service-to-UI adapter.
// The program reference will be set when Start() is called.
func NewServiceBridge(
	processMgr *processes.Manager,
	logStore *logs.Store,
	proxyServer *proxy.Server,
	eventBus *events.EventBus,
	testService TestManagementService,
	fileService FileBrowserService,
	settingsService SettingsService,
	inputService InputService,
	navigationService NavigationService,
	systemService SystemService,
	notificationsService NotificationsService,
) *ServiceBridge {
	sb := &ServiceBridge{
		processMgr:            processMgr,
		logStore:              logStore,
		proxyServer:           proxyServer,
		eventBus:              eventBus,
		testManagementService: testService,
		fileBrowserService:    fileService,
		settingsService:       settingsService,
		inputService:          inputService,
		navigationService:     navigationService,
		systemService:         systemService,
		notificationsService:  notificationsService,
		subscriptions:         []func(){},
	}
	// Initialize atomic.Value with empty holder
	sb.program.Store(&programHolder{prog: nil})
	return sb
}

// StartWithoutInitialMessages begins converting service events to BubbleTea messages.
// Does NOT send initial status updates to avoid deadlock during startup.
// The program parameter is used to send messages to the BubbleTea event loop.
//
// This must be called AFTER the tea.Program is created but BEFORE program.Run().
func (b *ServiceBridge) StartWithoutInitialMessages(program Program) {
	// Store program reference atomically (lock-free, thread-safe)
	b.program.Store(&programHolder{prog: program})

	// Subscribe to all service event sources
	b.subscribeToProcessManager()
	b.subscribeToLogStore()
	b.subscribeToEventBus()
	b.subscribeToProxyServer()
	b.subscribeToTestService()
	b.subscribeToFileBrowserService()
	b.subscribeToSettingsService()
	b.subscribeToInputService()
	b.subscribeToNavigationService()
	b.subscribeToSystemService()
	b.subscribeToNotificationsService()
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
	b.subscribeToTestService()
	b.subscribeToFileBrowserService()
	b.subscribeToSettingsService()
	b.subscribeToInputService()
	b.subscribeToNavigationService()
	b.subscribeToSystemService()
	b.subscribeToNotificationsService()

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

	// Process started events - single subscription
	unsubProcessStarted := b.eventBus.Subscribe(events.ProcessStarted, func(evt events.Event) {
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

	b.subscriptions = append(b.subscriptions, unsubProcessStarted)

	// Process exited events (map to ProcessStoppedMsg)
	unsubProcessExited := b.eventBus.Subscribe(events.ProcessExited, func(evt events.Event) {
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

	b.subscriptions = append(b.subscriptions, unsubProcessExited)

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

	// EventBus subscriptions now return unsubscribe functions for proper cleanup
	// Key subscriptions tracked above, others can be added as needed
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

// subscribeToAICoderService adds subscriptions for the new AICoderService events
// This method should be called after the AICoderService is created
func (b *ServiceBridge) SubscribeToAICoderService(aiCoderService interface{}) {
	if aiCoderService == nil || b.eventBus == nil {
		return
	}

	// Get the service's event bus
	// Use safe type assertion to check if service has EventBus() method
	type eventBusInterface interface{ EventBus() *events.EventBus }
	if svc, ok := aiCoderService.(eventBusInterface); ok {
		eventBus := svc.EventBus()

		// Subscribe to AI coder session events
		eventBus.Subscribe("aicoder.session.created", func(evt events.Event) {
			if prog := b.getProgram(); prog != nil {
				name := ""
				if n, ok := evt.Data["name"].(string); ok {
					name = n
				}
				prog.Send(messages.AICoderSessionCreatedMsg{
					SessionID: evt.ProcessID,
					Name:      name,
				})
			}
		})

		eventBus.Subscribe("aicoder.session.updated", func(evt events.Event) {
			if prog := b.getProgram(); prog != nil {
				status := messages.SessionActive // Default status
				lastActivity := time.Now()

				if s, ok := evt.Data["status"].(string); ok {
					status = messages.SessionStatus(s)
				}
				if la, ok := evt.Data["last_activity"].(time.Time); ok {
					lastActivity = la
				}

				prog.Send(messages.AICoderSessionUpdatedMsg{
					SessionID:    evt.ProcessID,
					Status:       status,
					LastActivity: lastActivity,
				})
			}
		})

		eventBus.Subscribe("aicoder.session.ended", func(evt events.Event) {
			if prog := b.getProgram(); prog != nil {
				reason := "stopped"
				if r, ok := evt.Data["reason"].(string); ok {
					reason = r
				}
				prog.Send(messages.AICoderSessionEndedMsg{
					SessionID: evt.ProcessID,
					Reason:    reason,
				})
			}
		})

		eventBus.Subscribe("aicoder.pty.output", func(evt events.Event) {
			if prog := b.getProgram(); prog != nil {
				data := []byte{}
				// Try both possible field names for compatibility
				if d, ok := evt.Data["data"].([]byte); ok {
					data = d
				} else if d, ok := evt.Data["output"].([]byte); ok {
					data = d
				}
				sequence := uint64(0)
				if s, ok := evt.Data["sequence"].(uint64); ok {
					sequence = s
				}

				// Convert PTY output to string for logging
				if len(data) > 0 && b.logStore != nil {
					sessionName := evt.ProcessID
					if name, ok := evt.Data["sessionName"].(string); ok {
						sessionName = name
					}
					outputStr := string(data)
					// Add PTY output to log store with AI coder context
					b.logStore.Add("ai-coder", sessionName, outputStr, false)
				}

				prog.Send(messages.AICoderPTYOutputMsg{
					SessionID: evt.ProcessID,
					Data:      data,
					Sequence:  sequence,
				})
			}
		})

		eventBus.Subscribe("aicoder.error", func(evt events.Event) {
			if prog := b.getProgram(); prog != nil {
				var err error
				if e, ok := evt.Data["error"].(error); ok {
					err = e
				}
				severity := "error"
				if s, ok := evt.Data["severity"].(string); ok {
					severity = s
				}

				prog.Send(messages.AICoderErrorMsg{
					SessionID: evt.ProcessID,
					Error:     err,
					Severity:  severity,
				})
			}
		})

		eventBus.Subscribe("aicoder.log", func(evt events.Event) {
			if prog := b.getProgram(); prog != nil {
				message := ""
				source := "ai-coder"
				if m, ok := evt.Data["message"].(string); ok {
					message = m
				}
				if s, ok := evt.Data["source"].(string); ok {
					source = s
				}

				// Add AI coder logs to the main log store
				if b.logStore != nil {
					b.logStore.Add("system", source, fmt.Sprintf("[AI-Coder] %s", message), false)
				}
			}
		})
	}
}

// Stop cleanly shuts down the ServiceBridge
func (b *ServiceBridge) Stop() {
	// Atomically clear program reference (lock-free, thread-safe)
	// This prevents observers from sending messages after shutdown
	b.program.Store(&programHolder{prog: nil})

	// Stop test management service
	if b.testManagementService != nil {
		b.testManagementService.Stop()
	}

	// Stop file browser service
	if b.fileBrowserService != nil {
		b.fileBrowserService.Stop()
	}

	// Stop settings service
	if b.settingsService != nil {
		b.settingsService.Stop()
	}

	// Stop input service
	if b.inputService != nil {
		b.inputService.Stop()
	}

	// Stop navigation service
	if b.navigationService != nil {
		b.navigationService.Stop()
	}

	// Stop system service
	if b.systemService != nil {
		b.systemService.Stop()
	}

	// Stop notifications service
	if b.notificationsService != nil {
		b.notificationsService.Stop()
	}

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

// subscribeToTestService converts test management service events to messages
func (b *ServiceBridge) subscribeToTestService() {
	if b.testManagementService == nil {
		return
	}

	// Start the test service event subscriptions
	b.testManagementService.SubscribeToEvents()

	// Test run started events
	b.eventBus.Subscribe("test.started", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			runID := ""
			framework := ""
			pattern := ""

			if id, ok := evt.Data["run_id"].(string); ok {
				runID = id
			}
			if f, ok := evt.Data["framework"].(string); ok {
				framework = f
			}
			if p, ok := evt.Data["pattern"].(string); ok {
				pattern = p
			}

			prog.Send(messages.TestRunStartedMsg{
				RunID:       runID,
				Framework:   framework,
				TestPattern: pattern,
				StartTime:   time.Now(),
			})
		}
	})

	// Test run completed events
	b.eventBus.Subscribe("test.completed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			runID := ""
			summary := interface{}(nil)

			if id, ok := evt.Data["run_id"].(string); ok {
				runID = id
			}
			if sum, ok := evt.Data["summary"]; ok {
				summary = sum
			}

			// Extract test summary from results if available
			var totalTests, passedTests, failedTests, skippedTests int
			var duration time.Duration
			if summaryMap, ok := summary.(map[string]interface{}); ok {
				if tt, ok := summaryMap["total"].(int); ok {
					totalTests = tt
				}
				if pt, ok := summaryMap["passed"].(int); ok {
					passedTests = pt
				}
				if ft, ok := summaryMap["failed"].(int); ok {
					failedTests = ft
				}
				if st, ok := summaryMap["skipped"].(int); ok {
					skippedTests = st
				}
				if d, ok := summaryMap["duration"].(time.Duration); ok {
					duration = d
				}
			}

			prog.Send(messages.TestRunCompletedMsg{
				RunID:        runID,
				TotalTests:   totalTests,
				PassedTests:  passedTests,
				FailedTests:  failedTests,
				SkippedTests: skippedTests,
				Duration:     duration,
				EndTime:      time.Now(),
			})
		}
	})

	// Test watch mode events
	b.eventBus.Subscribe("test.watch_started", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			// TODO: Extract watch pattern from event data
			prog.Send(messages.TestWatchModeStartedMsg{
				WatchPattern: "", // Extract from event data
				StartTime:    time.Now(),
			})
		}
	})

	b.eventBus.Subscribe("test.watch_stopped", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			prog.Send(messages.TestWatchModeStoppedMsg{
				EndTime: time.Now(),
			})
		}
	})

	// Test results cleared events
	b.eventBus.Subscribe("test.results_cleared", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			scope := ""
			if s, ok := evt.Data["scope"].(string); ok {
				scope = s
			}

			prog.Send(messages.TestResultsClearedMsg{
				RunID: scope, // Empty string = all cleared
			})
		}
	})

	// Test failure events (for detailed failure notifications)
	b.eventBus.Subscribe("test.failure", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			runID := evt.ProcessID
			testName := ""
			filePath := ""
			line := 0
			message := ""

			if name, ok := evt.Data["test_name"].(string); ok {
				testName = name
			}
			if path, ok := evt.Data["file_path"].(string); ok {
				filePath = path
			}
			if l, ok := evt.Data["line"].(int); ok {
				line = l
			}
			if msg, ok := evt.Data["message"].(string); ok {
				message = msg
			}

			// Create failure info for the test case
			failureInfo := &messages.TestFailureInfo{
				Message: message,
				Line:    line,
				// TODO: Add stack trace from event data if available
			}

			// Send test case completed with failure status
			prog.Send(messages.TestCaseCompletedMsg{
				RunID:       runID,
				TestName:    testName,
				FilePath:    filePath,
				Status:      "failed",
				Duration:    0, // TODO: Extract from event data if available
				FailureInfo: failureInfo,
				EndTime:     time.Now(),
			})
		}
	})
}

// subscribeToFileBrowserService converts file browser service events to messages
func (b *ServiceBridge) subscribeToFileBrowserService() {
	if b.fileBrowserService == nil {
		return
	}

	// Start the file browser service event subscriptions
	b.fileBrowserService.SubscribeToEvents()

	// File path changed events
	b.eventBus.Subscribe("file.path_changed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			path := ""
			if p, ok := evt.Data["path"].(string); ok {
				path = p
			}

			prog.Send(messages.FileBrowserPathChangedMsg{
				Path:      path,
				Timestamp: time.Now(),
			})
		}
	})

	// File system events
	b.eventBus.Subscribe("file.event", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			path := ""
			operation := ""
			timestamp := time.Now()
			fileInfo := interface{}(nil)

			if p, ok := evt.Data["path"].(string); ok {
				path = p
			}
			if op, ok := evt.Data["operation"].(string); ok {
				operation = op
			}
			if fi, ok := evt.Data["file_info"]; ok {
				fileInfo = fi
			}
			if ts, ok := evt.Data["timestamp"].(time.Time); ok {
				timestamp = ts
			}

			prog.Send(messages.FileBrowserFileEventMsg{
				Path:      path,
				Operation: operation,
				Timestamp: timestamp,
				FileInfo:  fileInfo,
			})
		}
	})

	// Watch started events
	b.eventBus.Subscribe("file.watch_started", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			path := ""
			if p, ok := evt.Data["path"].(string); ok {
				path = p
			}

			prog.Send(messages.FileBrowserWatchStartedMsg{
				Path:      path,
				Timestamp: time.Now(),
			})
		}
	})

	// Watch stopped events
	b.eventBus.Subscribe("file.watch_stopped", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			path := ""
			if p, ok := evt.Data["path"].(string); ok {
				path = p
			}

			prog.Send(messages.FileBrowserWatchStoppedMsg{
				Path:      path,
				Timestamp: time.Now(),
			})
		}
	})

	// File error events
	b.eventBus.Subscribe("file.error", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			errorMsg := ""
			operation := ""
			path := ""

			if err, ok := evt.Data["error"].(string); ok {
				errorMsg = err
			}
			if op, ok := evt.Data["operation"].(string); ok {
				operation = op
			}
			if p, ok := evt.Data["path"].(string); ok {
				path = p
			}

			prog.Send(messages.FileBrowserErrorMsg{
				Operation: operation,
				Path:      path,
				Error:     errorMsg,
				Timestamp: time.Now(),
			})
		}
	})
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

	// Check test management service status
	if b.testManagementService != nil {
		status := b.testManagementService.GetServiceStatus()
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: status.Service,
			Status:  status.Status,
			Details: status.Details,
		})
	}

	// Check file browser service status
	if b.fileBrowserService != nil {
		status := b.fileBrowserService.GetServiceStatus()
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: status.Service,
			Status:  status.Status,
			Details: status.Details,
		})
	}

	// Check settings service status
	if b.settingsService != nil {
		status := b.settingsService.GetServiceStatus()
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: status.Service,
			Status:  status.Status,
			Details: status.Details,
		})
	}

	// Check input service status
	if b.inputService != nil {
		status := b.inputService.GetServiceStatus()
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: status.Service,
			Status:  status.Status,
			Details: status.Details,
		})
	}

	// Check navigation service status
	if b.navigationService != nil {
		status := b.navigationService.GetServiceStatus()
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: status.Service,
			Status:  status.Status,
			Details: status.Details,
		})
	}

	// Check system service status
	if b.systemService != nil {
		status := b.systemService.GetServiceStatus()
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: status.Service,
			Status:  status.Status,
			Details: status.Details,
		})
	}

	// Check notifications service status
	if b.notificationsService != nil {
		status := b.notificationsService.GetServiceStatus()
		prog.Send(messages.ServiceStatusUpdateMsg{
			Service: status.Service,
			Status:  status.Status,
			Details: status.Details,
		})
	}
}

// subscribeToSettingsService converts settings service events to messages
func (b *ServiceBridge) subscribeToSettingsService() {
	if b.settingsService == nil {
		return
	}

	// Start the settings service event subscriptions
	b.settingsService.SubscribeToEvents()

	// Settings refreshed events
	b.eventBus.Subscribe("settings.refreshed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			if settingsInfo, ok := evt.Data["settings_info"]; ok {
				prog.Send(messages.SettingsRefreshedMsg{
					SettingsInfo: settingsInfo.(*messages.SettingsInfo),
					Timestamp:    time.Now(),
				})
			}
		}
	})

	// MCP installed events
	b.eventBus.Subscribe("settings.mcp_installed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			configFile := ""
			if cf, ok := evt.Data["config_file"].(string); ok {
				configFile = cf
			}
			prog.Send(messages.SettingsMCPInstalledMsg{
				ConfigFile: configFile,
				Timestamp:  time.Now(),
			})
		}
	})

	// Settings error events
	b.eventBus.Subscribe("settings.error", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			operation := ""
			errorMsg := ""
			if op, ok := evt.Data["operation"].(string); ok {
				operation = op
			}
			if err, ok := evt.Data["error"].(string); ok {
				errorMsg = err
			}
			prog.Send(messages.SettingsErrorMsg{
				Operation: operation,
				Error:     errorMsg,
				Timestamp: time.Now(),
			})
		}
	})
}

// subscribeToInputService converts input service events to messages
func (b *ServiceBridge) subscribeToInputService() {
	if b.inputService == nil {
		return
	}

	// Start the input service event subscriptions
	b.inputService.SubscribeToEvents()

	// Input configuration changed events
	b.eventBus.Subscribe("input.configuration_changed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			keyBindings := map[string]string{}

			if kb, ok := evt.Data["key_bindings"].(map[string]string); ok {
				keyBindings = kb
			}

			prog.Send(messages.KeyBindingChangedMsg{
				Action:  "configuration_changed",
				Key:     nil,
				Binding: keyBindings,
				Time:    time.Now(),
			})
		}
	})

	// Input focus changed events
	b.eventBus.Subscribe("input.focus.changed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			fromComponent := ""
			toComponent := ""

			if fc, ok := evt.Data["from_component"].(string); ok {
				fromComponent = fc
			}
			if tc, ok := evt.Data["to_component"].(string); ok {
				toComponent = tc
			}

			prog.Send(messages.InputFocusChangedMsg{
				FromComponent: fromComponent,
				ToComponent:   toComponent,
				Time:          time.Now(),
			})
		}
	})

	// Input processed events
	b.eventBus.Subscribe("input.processed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			key := interface{}(nil)
			context := ""
			handled := false

			if k, ok := evt.Data["key"].(interface{}); ok {
				key = k
			}
			if c, ok := evt.Data["context"].(string); ok {
				context = c
			}
			if h, ok := evt.Data["handled"].(bool); ok {
				handled = h
			}

			prog.Send(messages.InputKeyPressedMsg{
				Key:     key,
				Context: context,
				Handled: handled,
				Time:    time.Now(),
			})
		}
	})
}

// subscribeToNavigationService converts navigation service events to messages
func (b *ServiceBridge) subscribeToNavigationService() {
	if b.navigationService == nil {
		return
	}

	// Start the navigation service event subscriptions
	b.navigationService.SubscribeToEvents()

	// Navigation view changed events
	b.eventBus.Subscribe("navigation.view.changed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			fromView := ""
			toView := ""
			trigger := ""

			if fv, ok := evt.Data["from_view"].(string); ok {
				fromView = fv
			}
			if tv, ok := evt.Data["to_view"].(string); ok {
				toView = tv
			}
			if tr, ok := evt.Data["trigger"].(string); ok {
				trigger = tr
			}

			prog.Send(messages.ViewChangedMsg{
				FromView: messages.ViewType(fromView),
				ToView:   messages.ViewType(toView),
				Trigger:  trigger,
				Time:     time.Now(),
			})
		}
	})

	// Navigation view order changed events
	b.eventBus.Subscribe("navigation.view_order.changed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			newOrder := []messages.ViewType{}

			if no, ok := evt.Data["new_order"].([]interface{}); ok {
				for _, view := range no {
					if viewStr, ok := view.(string); ok {
						newOrder = append(newOrder, messages.ViewType(viewStr))
					}
				}
			}

			prog.Send(messages.ViewOrderChangedMsg{
				NewOrder: newOrder,
				Time:     time.Now(),
			})
		}
	})

	// Navigation number key mapping changed events
	b.eventBus.Subscribe("navigation.number_key_mapping.changed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			number := 0
			view := ""
			action := ""

			if n, ok := evt.Data["number"].(float64); ok {
				number = int(n)
			}
			if v, ok := evt.Data["view"].(string); ok {
				view = v
			}
			if a, ok := evt.Data["action"].(string); ok {
				action = a
			}

			prog.Send(messages.NumberKeyMappingChangedMsg{
				Number: number,
				View:   messages.ViewType(view),
				Action: action,
				Time:   time.Now(),
			})
		}
	})
}

// subscribeToSystemService converts system service events to messages
func (b *ServiceBridge) subscribeToSystemService() {
	if b.systemService == nil {
		return
	}

	// Start the system service event subscriptions
	b.systemService.SubscribeToEvents()

	// System message events
	b.eventBus.Subscribe("system.message", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			level := messages.InfoLevel
			content := ""
			source := ""

			if l, ok := evt.Data["level"].(string); ok {
				level = messages.LogLevel(l)
			}
			if c, ok := evt.Data["content"].(string); ok {
				content = c
			}
			if s, ok := evt.Data["source"].(string); ok {
				source = s
			}

			prog.Send(messages.SystemMessageAddedMsg{
				Level:       level,
				Content:     content,
				Source:      source,
				UnreadCount: 1, // Service will provide actual count
				Time:        time.Now(),
			})
		}
	})

	// System panel toggled events
	b.eventBus.Subscribe("system.panel_toggled", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			expanded := false
			trigger := ""

			if exp, ok := evt.Data["expanded"].(bool); ok {
				expanded = exp
			}
			if tr, ok := evt.Data["trigger"].(string); ok {
				trigger = tr
			}

			prog.Send(messages.SystemPanelToggledMsg{
				Expanded: expanded,
				Trigger:  trigger,
				Time:     time.Now(),
			})
		}
	})

	// System messages read events
	b.eventBus.Subscribe("system.messages_read", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			readCount := 0
			unreadCount := 0
			upToTime := time.Now()

			if rc, ok := evt.Data["read_count"].(int); ok {
				readCount = rc
			}
			if uc, ok := evt.Data["unread_count"].(int); ok {
				unreadCount = uc
			}
			if ut, ok := evt.Data["up_to_time"].(time.Time); ok {
				upToTime = ut
			}

			prog.Send(messages.SystemMessagesReadMsg{
				ReadCount:   readCount,
				UnreadCount: unreadCount,
				UpToTime:    upToTime,
				Time:        time.Now(),
			})
		}
	})

	// System buffer cleared events
	b.eventBus.Subscribe("system.buffer_cleared", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			clearedCount := 0
			trigger := ""

			if cc, ok := evt.Data["cleared_count"].(int); ok {
				clearedCount = cc
			}
			if tr, ok := evt.Data["trigger"].(string); ok {
				trigger = tr
			}

			prog.Send(messages.SystemMessageBufferClearedMsg{
				ClearedCount: clearedCount,
				Trigger:      trigger,
				Time:         time.Now(),
			})
		}
	})
}

// subscribeToNotificationsService converts notifications service events to messages
func (b *ServiceBridge) subscribeToNotificationsService() {
	if b.notificationsService == nil {
		return
	}

	// Start the notifications service event subscriptions
	b.notificationsService.SubscribeToEvents()

	// Notification shown events
	b.eventBus.Subscribe("notification.shown", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			notificationID := ""
			title := ""
			message := ""
			level := messages.InfoNotification
			duration := 5 * time.Second
			position := "top-right"

			if nid, ok := evt.Data["notification_id"].(string); ok {
				notificationID = nid
			}
			if t, ok := evt.Data["title"].(string); ok {
				title = t
			}
			if m, ok := evt.Data["message"].(string); ok {
				message = m
			}
			if l, ok := evt.Data["level"].(string); ok {
				level = messages.NotificationLevel(l)
			}
			if d, ok := evt.Data["duration"].(time.Duration); ok {
				duration = d
			}
			if p, ok := evt.Data["position"].(string); ok {
				position = p
			}

			prog.Send(messages.NotificationShownMsg{
				NotificationID: notificationID,
				Message:        title + ": " + message,
				Level:          level,
				Duration:       duration,
				Position:       position,
				Time:           time.Now(),
			})
		}
	})

	// Notification cleared events
	b.eventBus.Subscribe("notification.cleared", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			notificationID := ""
			trigger := ""

			if nid, ok := evt.Data["notification_id"].(string); ok {
				notificationID = nid
			}
			if tr, ok := evt.Data["trigger"].(string); ok {
				trigger = tr
			}

			prog.Send(messages.NotificationClearedMsg{
				NotificationID: notificationID,
				Trigger:        trigger,
				Time:           time.Now(),
			})
		}
	})

	// All notifications cleared events
	b.eventBus.Subscribe("notifications.all_cleared", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			clearedCount := 0
			trigger := ""

			if cc, ok := evt.Data["cleared_count"].(int); ok {
				clearedCount = cc
			}
			if tr, ok := evt.Data["trigger"].(string); ok {
				trigger = tr
			}

			prog.Send(messages.AllNotificationsClearedMsg{
				ClearedCount: clearedCount,
				Trigger:      trigger,
				Time:         time.Now(),
			})
		}
	})

	// Notification position changed events
	b.eventBus.Subscribe("notification.position_changed", func(evt events.Event) {
		if prog := b.getProgram(); prog != nil {
			oldPosition := ""
			newPosition := ""

			if op, ok := evt.Data["old_position"].(string); ok {
				oldPosition = op
			}
			if np, ok := evt.Data["new_position"].(string); ok {
				newPosition = np
			}

			prog.Send(messages.NotificationPositionChangedMsg{
				OldPosition: oldPosition,
				NewPosition: newPosition,
				Time:        time.Now(),
			})
		}
	})
}
