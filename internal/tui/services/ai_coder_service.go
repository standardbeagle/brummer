package services

// AI Coder service implementation for component mode.
// This provides a non-blocking, message-driven orchestration layer for AI agent
// interactions to avoid UI lockups and potential deadlocks. It intentionally
// avoids holding long-lived locks and communicates exclusively via service
// return values that the UI maps to messages.
// External dependencies: config for providers, logs store for user feedback.

import (
    "fmt"

    "github.com/standardbeagle/brummer/internal/config"
    "github.com/standardbeagle/brummer/internal/logs"
)

// AICoderServiceImpl implements AICoderService
// It currently computes a UI-agnostic start specification, leaving the UI to map
// it into component messages. Future: bridge to internal/aicoder manager.
type AICoderServiceImpl struct {
    cfg      *config.Config
    logStore *logs.Store
}

// NewAICoderService creates an AI coder service with loaded config and optional logs
func NewAICoderService(cfg *config.Config, logStore *logs.Store) *AICoderServiceImpl {
    if cfg == nil {
        // Load config if not provided; ignore error for best-effort defaults
        loaded, _ := config.Load()
        cfg = loaded
    }
    return &AICoderServiceImpl{cfg: cfg, logStore: logStore}
}

// Start returns a start spec for a new interactive AI session (non-blocking)
func (s *AICoderServiceImpl) Start(provider string) (AICoderStartSpec, error) {
    if s.logStore != nil {
        s.logStore.Add("system", "System", fmt.Sprintf("🤖 AI: starting provider '%s'", provider), false)
    }

    // Compute defaults from configuration
    model := ""
    if s.cfg != nil && s.cfg.AICoders != nil && s.cfg.AICoders.Providers != nil {
        if pcfg, ok := s.cfg.AICoders.Providers[provider]; ok && pcfg != nil {
            model = pcfg.GetModel()
        }
    }

    return AICoderStartSpec{
        Name:         fmt.Sprintf("%s session", provider),
        Model:        model,
        SystemPrompt: "You are a helpful AI coding assistant.",
        WorkingDir:   ".",
    }, nil
}

// Stop terminates a session (no-op placeholder for now)
func (s *AICoderServiceImpl) Stop(sessionID string) error {
    if s.logStore != nil {
        s.logStore.Add("system", "System", fmt.Sprintf("🛑 Stopping AI session %s", sessionID), false)
    }
    return nil
}

// ListSessions returns known sessions; UI keeps authoritative list right now
func (s *AICoderServiceImpl) ListSessions() []AISession {
    // Without a backing manager, return empty. The component/view maintains its state.
    return nil
}

// GetProviders returns provider names from config (with fallback to defaults)
func (s *AICoderServiceImpl) GetProviders() []string {
    names := make([]string, 0)
    if s.cfg != nil && s.cfg.AICoders != nil && s.cfg.AICoders.Providers != nil {
        for name := range s.cfg.AICoders.Providers {
            names = append(names, name)
        }
    }
    if len(names) == 0 {
        // Fallback to built-in defaults if none configured
        defaults := config.GetDefaultProviderConfigs()
        for name := range defaults {
            names = append(names, name)
        }
    }
    return names
}

