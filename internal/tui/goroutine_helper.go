package tui

import (
	"fmt"
	"runtime/debug"
)

// SafeGoroutine executes a function in a goroutine with panic recovery and error context
func SafeGoroutine(operation string, fn func() error, onError func(error)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				err := fmt.Errorf("panic in goroutine %s: %v\nStack trace:\n%s", operation, r, string(stack))
				if onError != nil {
					onError(err)
				}
			}
		}()

		if err := fn(); err != nil && onError != nil {
			onError(fmt.Errorf("error in %s: %w", operation, err))
		}
	}()
}

// SafeGoroutineNoError executes a function in a goroutine with panic recovery (for functions that don't return errors)
func SafeGoroutineNoError(operation string, fn func(), onPanic func(error)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				err := fmt.Errorf("panic in goroutine %s: %v\nStack trace:\n%s", operation, r, string(stack))
				if onPanic != nil {
					onPanic(err)
				}
			}
		}()

		fn()
	}()
}
