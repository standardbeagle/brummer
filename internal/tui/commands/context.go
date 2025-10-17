package commands

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// CommandContext provides execution context for commands
type CommandContext struct {
	// Go context for cancellation and timeout
	ctx context.Context

	// Operation context for metadata
	operationContext OperationContext

	// Cancellation function
	cancel context.CancelFunc
}

// NewCommandContext creates a new command context with timeout
func NewCommandContext(opContext OperationContext) *CommandContext {
	ctx, cancel := context.WithTimeout(context.Background(), opContext.Timeout)
	return &CommandContext{
		ctx:              ctx,
		operationContext: opContext,
		cancel:           cancel,
	}
}

// NewCommandContextWithParent creates a new command context with a parent context
func NewCommandContextWithParent(parent context.Context, opContext OperationContext) *CommandContext {
	ctx, cancel := context.WithTimeout(parent, opContext.Timeout)
	return &CommandContext{
		ctx:              ctx,
		operationContext: opContext,
		cancel:           cancel,
	}
}

// Context returns the Go context
func (c *CommandContext) Context() context.Context {
	return c.ctx
}

// OperationContext returns the operation context
func (c *CommandContext) OperationContext() OperationContext {
	return c.operationContext
}

// Done returns a channel that's closed when the context is cancelled
func (c *CommandContext) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns the context error if any
func (c *CommandContext) Err() error {
	return c.ctx.Err()
}

// Deadline returns the deadline for the context
func (c *CommandContext) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

// IsCancelled returns true if the context was cancelled
func (c *CommandContext) IsCancelled() bool {
	return c.ctx.Err() == context.Canceled
}

// IsTimedOut returns true if the context timed out
func (c *CommandContext) IsTimedOut() bool {
	return c.ctx.Err() == context.DeadlineExceeded
}

// Cancel cancels the context
func (c *CommandContext) Cancel() {
	c.cancel()
}

// Cleanup releases resources associated with the context
func (c *CommandContext) Cleanup() {
	c.cancel()
}

// TimeRemaining returns the time remaining before timeout
func (c *CommandContext) TimeRemaining() time.Duration {
	deadline, ok := c.ctx.Deadline()
	if !ok {
		return c.operationContext.Timeout // Fallback to configured timeout
	}
	return time.Until(deadline)
}

// IsExpired returns true if the context has expired
func (c *CommandContext) IsExpired() bool {
	return c.TimeRemaining() <= 0
}

// WithValue returns a copy of the context with the given value
func (c *CommandContext) WithValue(key, value interface{}) *CommandContext {
	return &CommandContext{
		ctx:              context.WithValue(c.ctx, key, value),
		operationContext: c.operationContext,
		cancel:           c.cancel,
	}
}

// Value returns the value associated with key from the context
func (c *CommandContext) Value(key interface{}) interface{} {
	return c.ctx.Value(key)
}

// String returns a string representation of the context
func (c *CommandContext) String() string {
	return fmt.Sprintf("CommandContext{timeout=%v, operationType=%s, version=%d}",
		c.operationContext.Timeout, c.operationContext.OperationType, c.operationContext.Version)
}

// ContextKey type for context values
type ContextKey string

// Common context keys
const (
	ContextKeyRequestID ContextKey = "request_id"
	ContextKeyUserID    ContextKey = "user_id"
	ContextKeyTraceID   ContextKey = "trace_id"
	ContextKeySpanID    ContextKey = "span_id"
	ContextKeyMetadata  ContextKey = "metadata"
)

// WithRequestID adds a request ID to the context
func (c *CommandContext) WithRequestID(requestID string) *CommandContext {
	return c.WithValue(ContextKeyRequestID, requestID)
}

// WithUserID adds a user ID to the context
func (c *CommandContext) WithUserID(userID string) *CommandContext {
	return c.WithValue(ContextKeyUserID, userID)
}

// WithTraceID adds a trace ID to the context
func (c *CommandContext) WithTraceID(traceID string) *CommandContext {
	return c.WithValue(ContextKeyTraceID, traceID)
}

// WithSpanID adds a span ID to the context
func (c *CommandContext) WithSpanID(spanID string) *CommandContext {
	return c.WithValue(ContextKeySpanID, spanID)
}

// WithMetadata adds metadata to the context
func (c *CommandContext) WithMetadata(metadata map[string]interface{}) *CommandContext {
	return c.WithValue(ContextKeyMetadata, metadata)
}

// GetRequestID retrieves the request ID from the context
func (c *CommandContext) GetRequestID() string {
	if val, ok := c.Value(ContextKeyRequestID).(string); ok {
		return val
	}
	return ""
}

// GetUserID retrieves the user ID from the context
func (c *CommandContext) GetUserID() string {
	if val, ok := c.Value(ContextKeyUserID).(string); ok {
		return val
	}
	return ""
}

// GetTraceID retrieves the trace ID from the context
func (c *CommandContext) GetTraceID() string {
	if val, ok := c.Value(ContextKeyTraceID).(string); ok {
		return val
	}
	return ""
}

// GetSpanID retrieves the span ID from the context
func (c *CommandContext) GetSpanID() string {
	if val, ok := c.Value(ContextKeySpanID).(string); ok {
		return val
	}
	return ""
}

// GetMetadata retrieves metadata from the context
func (c *CommandContext) GetMetadata() map[string]interface{} {
	if val, ok := c.Value(ContextKeyMetadata).(map[string]interface{}); ok {
		return val
	}
	return make(map[string]interface{})
}

// ExecuteWithTimeout executes a function with the context's timeout using async commands
func (c *CommandContext) ExecuteWithTimeout(fn func() (interface{}, error)) (interface{}, error) {
	// Create async context command
	asyncCommands := NewContextAsyncCommands()
	cmd := asyncCommands.ExecuteWithTimeout(
		fmt.Sprintf("ctx-%d", time.Now().UnixNano()),
		"timeout_operation",
		c.TimeRemaining(),
		nil, // model not needed for this operation
	)

	// Execute the command and get result
	result := cmd()

	// Extract the actual result from the command result
	if msg, ok := result.(tea.Msg); ok {
		// Handle command completion message
		if completionMsg, ok := msg.(*messages.CommandCompletedMessage); ok {
			if completionMsg.Result != nil {
				return completionMsg.Result, nil
			}
		}
		// Handle command failed message
		if failedMsg, ok := msg.(*messages.CommandFailedMessage); ok {
			return nil, failedMsg.Error
		}
	}

	// Fallback to direct execution if command pattern fails
	// Use synchronous execution to avoid goroutines
	value, err := fn()
	if err != nil {
		return nil, err
	}
	return value, nil
}

// ContextPool represents a pool of contexts for reuse
type ContextPool struct {
	pool chan *CommandContext
	factory func(OperationContext) *CommandContext
}

// NewContextPool creates a new context pool
func NewContextPool(size int, factory func(OperationContext) *CommandContext) *ContextPool {
	return &ContextPool{
		pool: make(chan *CommandContext, size),
		factory: factory,
	}
}

// Get retrieves a context from the pool or creates a new one
func (p *ContextPool) Get(opContext OperationContext) *CommandContext {
	select {
	case ctx := <-p.pool:
		// Reset the context with new operation context
		ctx.operationContext = opContext
		return ctx
	default:
		return p.factory(opContext)
	}
}

// Put returns a context to the pool for reuse
func (p *ContextPool) Put(ctx *CommandContext) {
	select {
	case p.pool <- ctx:
		// Context returned to pool
	default:
		// Pool is full, cleanup and discard
		ctx.Cleanup()
	}
}

// Close closes the context pool and cleans up all contexts
func (p *ContextPool) Close() {
	close(p.pool)
	for ctx := range p.pool {
		ctx.Cleanup()
	}
}