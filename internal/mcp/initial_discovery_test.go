package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialDiscovery(t *testing.T) {
	// Create temp directory for instances
	tempDir := t.TempDir()
	instancesDir := filepath.Join(tempDir, "instances")
	require.NoError(t, os.MkdirAll(instancesDir, 0755))

	// Create an instance file BEFORE starting discovery
	instance := &discovery.Instance{
		ID:        "pre-existing-instance",
		Name:      "Test Pre-existing",
		Directory: "/test/pre",
		Port:      9999,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	instance.ProcessInfo.PID = 12345
	instance.ProcessInfo.Executable = "brum"

	require.NoError(t, discovery.RegisterInstance(instancesDir, instance))

	// Now simulate the hub startup sequence
	connMgr := NewConnectionManager()
	defer connMgr.Stop()

	// Create discovery system (this does initial scan)
	disc, err := discovery.New(instancesDir)
	require.NoError(t, err)
	defer disc.Stop()

	// Track discovered instances
	discoveredInstances := make(map[string]bool)

	// Register callback
	disc.OnUpdate(func(instances map[string]*discovery.Instance) {
		for _, inst := range instances {
			discoveredInstances[inst.ID] = true
			connMgr.RegisterInstance(inst)
		}
	})

	// Start discovery
	disc.Start()

	// Process existing instances (this is our fix)
	existingInstances := disc.GetInstances()
	for _, inst := range existingInstances {
		discoveredInstances[inst.ID] = true
		connMgr.RegisterInstance(inst)
	}

	// Verify the pre-existing instance was registered immediately
	connections := connMgr.ListInstances()
	require.Len(t, connections, 1, "Pre-existing instance should be registered immediately")
	assert.Equal(t, "pre-existing-instance", connections[0].InstanceID)

	// Now add a new instance after discovery started
	newInstance := &discovery.Instance{
		ID:        "new-instance",
		Name:      "Test New",
		Directory: "/test/new",
		Port:      8888,
		StartedAt: time.Now(),
		LastPing:  time.Now(),
	}
	newInstance.ProcessInfo.PID = 54321
	newInstance.ProcessInfo.Executable = "brum"

	require.NoError(t, discovery.RegisterInstance(instancesDir, newInstance))

	// Wait for file watcher to pick it up
	time.Sleep(100 * time.Millisecond)

	// Verify both instances are now registered
	connections = connMgr.ListInstances()
	assert.Len(t, connections, 2, "Both instances should be registered")

	// Verify we discovered both
	assert.True(t, discoveredInstances["pre-existing-instance"], "Pre-existing instance should be discovered")
	assert.True(t, discoveredInstances["new-instance"], "New instance should be discovered")
}
