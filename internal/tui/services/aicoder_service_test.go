package services

import (
	"context"
	"testing"
	"time"
)

// TestNewAICoderService verifies that the service initializes with empty registry
func TestNewAICoderService(t *testing.T) {
	// This test will fail until we implement NewAICoderService
	ctx := context.Background()
	service := NewAICoderService(ctx)

	if service == nil {
		t.Fatal("NewAICoderService returned nil")
	}

	// Verify empty session registry
	sessions := service.ListSessions()
	if len(sessions) != 0 {
		t.Errorf("Expected empty session list, got %d sessions", len(sessions))
	}
}

// TestServiceEventBusReference verifies EventBus is valid
func TestServiceEventBusReference(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	if service == nil {
		t.Fatal("NewAICoderService returned nil")
	}

	// EventBus should be accessible
	eventBus := service.EventBus()
	if eventBus == nil {
		t.Error("EventBus() returned nil")
	}
}

// TestServiceShutdownChannel verifies shutdown channel created
func TestServiceShutdownChannel(t *testing.T) {
	ctx := context.Background()
	service := NewAICoderService(ctx)

	if service == nil {
		t.Fatal("NewAICoderService returned nil")
	}

	// Shutdown should complete without hanging
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.Shutdown(shutdownCtx)
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}
