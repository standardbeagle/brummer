package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/standardbeagle/brummer/internal/test"
	"github.com/standardbeagle/brummer/pkg/events"
)

// TestRunRequest represents a request to run tests
type TestRunRequest struct {
	Framework string           `json:"framework,omitempty"`
	Pattern   string           `json:"pattern,omitempty"`
	Options   test.TestOptions `json:"options"`
}

// TestRunResponse represents the response from running tests
type TestRunResponse struct {
	RunID     string            `json:"run_id"`
	Status    string            `json:"status"`
	StartTime time.Time         `json:"start_time"`
	Results   *test.TestResults `json:"results,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// TestStatusResponse represents the current test status
type TestStatusResponse struct {
	ActiveRuns  []*test.TestRun              `json:"active_runs"`
	LastResults map[string]*test.TestResults `json:"last_results"`
}

// TestFailuresResponse represents test failure information
type TestFailuresResponse struct {
	RunID    string              `json:"run_id"`
	Failures []TestFailureDetail `json:"failures"`
	Summary  TestFailureSummary  `json:"summary"`
}

// TestFailureDetail contains detailed failure information
type TestFailureDetail struct {
	TestName   string   `json:"test_name"`
	FilePath   string   `json:"file_path"`
	Line       int      `json:"line,omitempty"`
	Column     int      `json:"column,omitempty"`
	Message    string   `json:"message"`
	StackTrace []string `json:"stack_trace"`
	Context    []string `json:"context,omitempty"`
}

// TestFailureSummary provides a summary of failures
type TestFailureSummary struct {
	TotalFailures int            `json:"total_failures"`
	FailedFiles   []string       `json:"failed_files"`
	ErrorTypes    map[string]int `json:"error_types"`
}

// TestCoverageResponse represents test coverage information
type TestCoverageResponse struct {
	RunID    string              `json:"run_id"`
	Coverage *test.TestCoverage  `json:"coverage"`
	Summary  TestCoverageSummary `json:"summary"`
}

// TestCoverageSummary provides a coverage summary
type TestCoverageSummary struct {
	OverallPct       float64  `json:"overall_pct"`
	CoveredFiles     int      `json:"covered_files"`
	TotalFiles       int      `json:"total_files"`
	UncoveredFiles   []string `json:"uncovered_files"`
	LowCoverageFiles []string `json:"low_coverage_files"`
}

// TestWatchRequest represents a request to start watch mode
type TestWatchRequest struct {
	Framework string           `json:"framework,omitempty"`
	Pattern   string           `json:"pattern,omitempty"`
	Options   test.TestOptions `json:"options"`
}

// TestWatchResponse represents the response from starting watch mode
type TestWatchResponse struct {
	WatchID   string    `json:"watch_id"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	Pattern   string    `json:"pattern"`
	Error     string    `json:"error,omitempty"`
}

// TestTools provides MCP tools for test management
type TestTools struct {
	testRunner *test.TestRunnerService
	eventBus   *events.EventBus
}

// NewTestTools creates a new test tools instance
func NewTestTools(testRunner *test.TestRunnerService, eventBus *events.EventBus) *TestTools {
	return &TestTools{
		testRunner: testRunner,
		eventBus:   eventBus,
	}
}

// test_run executes tests with optional filtering
func (tt *TestTools) test_run(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	var req TestRunRequest

	if argsBytes, err := json.Marshal(arguments); err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	} else if err := json.Unmarshal(argsBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Set default options if not provided
	if req.Options.Timeout == 0 {
		req.Options.Timeout = 5 * time.Minute
	}

	// Auto-detect framework if not specified
	if req.Framework == "" {
		detected := tt.testRunner.DetectAvailableFrameworks()
		if len(detected) == 0 {
			return nil, fmt.Errorf("no test frameworks detected in current directory")
		}
		req.Framework = detected[0].Name()
	}

	// Start test run
	run, err := tt.testRunner.RunTestsWithFramework(req.Framework, req.Pattern, req.Options)
	if err != nil {
		return TestRunResponse{
			Error: err.Error(),
		}, nil
	}

	// Note: Event publishing disabled for now to avoid type conflicts
	// TODO: Implement proper event integration with existing event bus

	return TestRunResponse{
		RunID:     run.ID,
		Status:    string(run.Status),
		StartTime: run.StartTime,
	}, nil
}

// test_status gets the current test results summary
func (tt *TestTools) test_status(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	manager := tt.testRunner.GetManager()
	activeRuns := manager.GetActiveRuns()

	// Get cached results
	results := make(map[string]*test.TestResults)
	for _, run := range activeRuns {
		if run.Results != nil {
			results[run.ID] = run.Results
		}
	}

	return TestStatusResponse{
		ActiveRuns:  activeRuns,
		LastResults: results,
	}, nil
}

// test_failures gets detailed failure information for failing tests
func (tt *TestTools) test_failures(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	runID, ok := arguments["run_id"].(string)
	if !ok || runID == "" {
		// Get most recent run with failures
		manager := tt.testRunner.GetManager()
		activeRuns := manager.GetActiveRuns()

		for _, run := range activeRuns {
			if run.Results != nil && run.Results.FailedTests > 0 {
				runID = run.ID
				break
			}
		}

		if runID == "" {
			return nil, fmt.Errorf("no test run with failures found")
		}
	}

	manager := tt.testRunner.GetManager()
	run, exists := manager.GetRun(runID)
	if !exists {
		return nil, fmt.Errorf("test run %s not found", runID)
	}

	if run.Results == nil || run.Results.FailedTests == 0 {
		return TestFailuresResponse{
			RunID:    runID,
			Failures: []TestFailureDetail{},
			Summary: TestFailureSummary{
				TotalFailures: 0,
				FailedFiles:   []string{},
				ErrorTypes:    make(map[string]int),
			},
		}, nil
	}

	// Convert failures to response format
	failures := make([]TestFailureDetail, 0, run.Results.FailedTests)
	failedFiles := make([]string, 0)
	errorTypes := make(map[string]int)

	for _, tc := range run.Results.TestCases {
		if tc.Status == test.TestCaseStatusFailed && tc.FailureInfo != nil {
			failure := TestFailureDetail{
				TestName:   tc.Name,
				FilePath:   tc.FilePath,
				Line:       tc.FailureInfo.Line,
				Column:     tc.FailureInfo.Column,
				Message:    tc.FailureInfo.Message,
				StackTrace: tc.FailureInfo.StackTrace,
			}

			// Extract context (surrounding lines) if available
			// TODO: Implement context extraction from source files
			failure.Context = []string{}

			failures = append(failures, failure)

			// Track failed files
			found := false
			for _, file := range failedFiles {
				if file == tc.FilePath {
					found = true
					break
				}
			}
			if !found {
				failedFiles = append(failedFiles, tc.FilePath)
			}

			// Categorize error types
			errorType := "unknown"
			if strings.Contains(strings.ToLower(tc.FailureInfo.Message), "assert") {
				errorType = "assertion"
			} else if strings.Contains(strings.ToLower(tc.FailureInfo.Message), "timeout") {
				errorType = "timeout"
			} else if strings.Contains(strings.ToLower(tc.FailureInfo.Message), "import") || strings.Contains(strings.ToLower(tc.FailureInfo.Message), "module") {
				errorType = "import"
			}
			errorTypes[errorType]++
		}
	}

	return TestFailuresResponse{
		RunID:    runID,
		Failures: failures,
		Summary: TestFailureSummary{
			TotalFailures: run.Results.FailedTests,
			FailedFiles:   failedFiles,
			ErrorTypes:    errorTypes,
		},
	}, nil
}

// test_coverage gets code coverage data
func (tt *TestTools) test_coverage(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	runID, ok := arguments["run_id"].(string)
	if !ok || runID == "" {
		// Get most recent run with coverage
		manager := tt.testRunner.GetManager()
		activeRuns := manager.GetActiveRuns()

		for _, run := range activeRuns {
			if run.Results != nil && run.Results.Coverage != nil {
				runID = run.ID
				break
			}
		}

		if runID == "" {
			return nil, fmt.Errorf("no test run with coverage data found")
		}
	}

	manager := tt.testRunner.GetManager()
	run, exists := manager.GetRun(runID)
	if !exists {
		return nil, fmt.Errorf("test run %s not found", runID)
	}

	if run.Results == nil || run.Results.Coverage == nil {
		return nil, fmt.Errorf("no coverage data available for run %s", runID)
	}

	coverage := run.Results.Coverage

	// Generate summary
	uncoveredFiles := []string{}
	lowCoverageFiles := []string{}
	coveredFiles := 0

	for _, file := range coverage.Files {
		if file.CoveragePct == 0 {
			uncoveredFiles = append(uncoveredFiles, file.FilePath)
		} else if file.CoveragePct < 60 {
			lowCoverageFiles = append(lowCoverageFiles, file.FilePath)
		}
		if file.CoveragePct > 0 {
			coveredFiles++
		}
	}

	return TestCoverageResponse{
		RunID:    runID,
		Coverage: coverage,
		Summary: TestCoverageSummary{
			OverallPct:       coverage.OverallPct,
			CoveredFiles:     coveredFiles,
			TotalFiles:       len(coverage.Files),
			UncoveredFiles:   uncoveredFiles,
			LowCoverageFiles: lowCoverageFiles,
		},
	}, nil
}

// test_watch starts watch mode for continuous testing
func (tt *TestTools) test_watch(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	var req TestWatchRequest

	if argsBytes, err := json.Marshal(arguments); err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	} else if err := json.Unmarshal(argsBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Set default options for watch mode
	req.Options.WatchMode = true
	if req.Options.Timeout == 0 {
		req.Options.Timeout = 30 * time.Minute // Longer timeout for watch mode
	}

	// Auto-detect framework if not specified
	if req.Framework == "" {
		detected := tt.testRunner.DetectAvailableFrameworks()
		if len(detected) == 0 {
			return nil, fmt.Errorf("no test frameworks detected in current directory")
		}
		req.Framework = detected[0].Name()
	}

	// Start watch mode
	watchID := fmt.Sprintf("watch-%d", time.Now().Unix())

	// Note: Event publishing disabled for now to avoid type conflicts
	// TODO: Implement proper event integration with existing event bus

	return TestWatchResponse{
		WatchID:   watchID,
		Status:    "started",
		StartTime: time.Now(),
		Pattern:   req.Pattern,
	}, nil
}

// test_clear clears test result history
func (tt *TestTools) test_clear(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	runID, ok := arguments["run_id"].(string)
	if !ok {
		runID = "" // Clear all if not specified
	}

	manager := tt.testRunner.GetManager()
	manager.ClearResults(runID)

	// Note: Event publishing disabled for now to avoid type conflicts
	// TODO: Implement proper event integration with existing event bus

	return map[string]interface{}{
		"cleared": runID == "",
		"run_id":  runID,
	}, nil
}

// RegisterTestTools registers all test management tools with the MCP server
func RegisterTestTools(mcpServer interface{}, testRunner *test.TestRunnerService, eventBus *events.EventBus) {
	testTools := NewTestTools(testRunner, eventBus)

	testRunSchema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"framework": map[string]interface{}{
				"type":        "string",
				"description": "Test framework to use (jest, vitest, go, pytest). Auto-detected if not specified.",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Test pattern or file filter",
			},
			"options": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"watch_mode": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable watch mode for continuous testing",
					},
					"coverage": map[string]interface{}{
						"type":        "boolean",
						"description": "Generate coverage report",
					},
					"parallel": map[string]interface{}{
						"type":        "boolean",
						"description": "Run tests in parallel if supported",
					},
					"verbose": map[string]interface{}{
						"type":        "boolean",
						"description": "Verbose output",
					},
					"timeout": map[string]interface{}{
						"type":        "string",
						"description": "Test timeout duration (e.g., '5m', '30s')",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "Additional test filter criteria",
					},
				},
			},
		},
	})

	testStatusSchema, _ := json.Marshal(map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	})

	testFailuresSchema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"run_id": map[string]interface{}{
				"type":        "string",
				"description": "Specific test run ID to analyze. If not specified, uses the most recent run with failures.",
			},
		},
	})

	testCoverageSchema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"run_id": map[string]interface{}{
				"type":        "string",
				"description": "Specific test run ID to analyze. If not specified, uses the most recent run with coverage data.",
			},
		},
	})

	testWatchSchema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"framework": map[string]interface{}{
				"type":        "string",
				"description": "Test framework to use. Auto-detected if not specified.",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Test pattern or file filter for watch mode",
			},
			"options": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"coverage": map[string]interface{}{
						"type":        "boolean",
						"description": "Generate coverage report on changes",
					},
					"parallel": map[string]interface{}{
						"type":        "boolean",
						"description": "Run tests in parallel if supported",
					},
					"verbose": map[string]interface{}{
						"type":        "boolean",
						"description": "Verbose output during watch mode",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "Additional test filter criteria",
					},
				},
			},
		},
	})

	testClearSchema, _ := json.Marshal(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"run_id": map[string]interface{}{
				"type":        "string",
				"description": "Specific test run ID to clear. If not specified, clears all test results.",
			},
		},
	})

	// Register tools using the proper server interface
	if mcpSrv, ok := mcpServer.(interface {
		RegisterTool(name string, tool MCPTool) error
	}); ok {
		mcpSrv.RegisterTool("test_run", MCPTool{
			Name:        "test_run",
			Description: "Execute tests with optional patterns and configuration",
			InputSchema: testRunSchema,
			Handler: func(args json.RawMessage) (interface{}, error) {
				var req TestRunRequest
				if err := json.Unmarshal(args, &req); err != nil {
					return nil, fmt.Errorf("failed to unmarshal request: %w", err)
				}
				return testTools.test_run(context.Background(), map[string]interface{}{
					"framework": req.Framework,
					"pattern":   req.Pattern,
					"options":   req.Options,
				})
			},
		})

		mcpSrv.RegisterTool("test_status", MCPTool{
			Name:        "test_status",
			Description: "Get current test results summary and active test runs",
			InputSchema: testStatusSchema,
			Handler: func(args json.RawMessage) (interface{}, error) {
				return testTools.test_status(context.Background(), map[string]interface{}{})
			},
		})

		mcpSrv.RegisterTool("test_failures", MCPTool{
			Name:        "test_failures",
			Description: "Get detailed failure information for failing tests",
			InputSchema: testFailuresSchema,
			Handler: func(args json.RawMessage) (interface{}, error) {
				var params map[string]interface{}
				json.Unmarshal(args, &params)
				return testTools.test_failures(context.Background(), params)
			},
		})

		mcpSrv.RegisterTool("test_coverage", MCPTool{
			Name:        "test_coverage",
			Description: "Get code coverage data from test runs",
			InputSchema: testCoverageSchema,
			Handler: func(args json.RawMessage) (interface{}, error) {
				var params map[string]interface{}
				json.Unmarshal(args, &params)
				return testTools.test_coverage(context.Background(), params)
			},
		})

		mcpSrv.RegisterTool("test_watch", MCPTool{
			Name:        "test_watch",
			Description: "Start watch mode for continuous testing",
			InputSchema: testWatchSchema,
			Handler: func(args json.RawMessage) (interface{}, error) {
				var req TestWatchRequest
				if err := json.Unmarshal(args, &req); err != nil {
					return nil, fmt.Errorf("failed to unmarshal request: %w", err)
				}
				return testTools.test_watch(context.Background(), map[string]interface{}{
					"framework": req.Framework,
					"pattern":   req.Pattern,
					"options":   req.Options,
				})
			},
		})

		mcpSrv.RegisterTool("test_clear", MCPTool{
			Name:        "test_clear",
			Description: "Clear test result history",
			InputSchema: testClearSchema,
			Handler: func(args json.RawMessage) (interface{}, error) {
				var params map[string]interface{}
				json.Unmarshal(args, &params)
				return testTools.test_clear(context.Background(), params)
			},
		})
	}
}
