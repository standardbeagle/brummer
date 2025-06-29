package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ResourceInfo represents information about a resource from ListResources response
type ResourceInfo struct {
	URI         string          `json:"uri"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	MimeType    string          `json:"mimeType,omitempty"`
}

// ResourceWithHandler extends Resource with a handler function
type ResourceWithHandler struct {
	Resource
	Handler func() (interface{}, error)
}

// ProxyResource creates a proxy Resource that forwards reads to a connected instance
func ProxyResource(instanceID string, resourceInfo ResourceInfo, connMgr *ConnectionManager) ResourceWithHandler {
	// Prefix the URI with instance ID
	prefixedURI := fmt.Sprintf("%s_%s", instanceID, resourceInfo.URI)
	
	return ResourceWithHandler{
		Resource: Resource{
			URI:         prefixedURI,
			Name:        fmt.Sprintf("[%s] %s", instanceID, resourceInfo.Name),
			Description: fmt.Sprintf("[%s] %s", instanceID, resourceInfo.Description),
			MimeType:    resourceInfo.MimeType,
		},
		Handler: func() (interface{}, error) {
			// Get the client for this instance
			connections := connMgr.ListInstances()
			
			var activeClient *HubClient
			for _, conn := range connections {
				if conn.InstanceID == instanceID && conn.State == StateActive && conn.Client != nil {
					activeClient = conn.Client
					break
				}
			}
			
			if activeClient == nil {
				return nil, fmt.Errorf("instance %s is not connected or has no active client", instanceID)
			}
			
			// Forward the resource read to the instance with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			result, err := activeClient.ReadResource(ctx, resourceInfo.URI)
			if err != nil {
				return nil, fmt.Errorf("resource read failed: %w", err)
			}
			
			// Parse the result
			var response interface{}
			if err := json.Unmarshal(result, &response); err != nil {
				// Return raw result if parsing fails
				return result, nil
			}
			
			return response, nil
		},
	}
}

// ExtractInstanceAndResource parses a prefixed resource URI to get instance ID and original URI
func ExtractInstanceAndResource(prefixedURI string) (instanceID, resourceURI string, err error) {
	parts := strings.SplitN(prefixedURI, "_", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid resource URI format: expected 'instanceID_uri', got '%s'", prefixedURI)
	}
	return parts[0], parts[1], nil
}

// RegisterInstanceResources fetches and registers resources from a connected instance
func RegisterInstanceResources(server *StreamableServer, connMgr *ConnectionManager, instanceID string) error {
	// Find the connection for the instance
	connections := connMgr.ListInstances()
	
	var activeClient *HubClient
	for _, conn := range connections {
		if conn.InstanceID == instanceID && conn.State == StateActive && conn.Client != nil {
			activeClient = conn.Client
			break
		}
	}
	
	if activeClient == nil {
		return fmt.Errorf("no active client available for instance %s", instanceID)
	}
	
	// List resources from the instance
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	resourcesData, err := activeClient.ListResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to list resources from instance %s: %w", instanceID, err)
	}
	
	// Parse the resources response
	var resourcesResponse struct {
		Resources []ResourceInfo `json:"resources"`
	}
	if err := json.Unmarshal(resourcesData, &resourcesResponse); err != nil {
		return fmt.Errorf("failed to parse resources response: %w", err)
	}
	
	// Convert and register each resource
	var resources []ResourceWithHandler
	for _, resourceInfo := range resourcesResponse.Resources {
		proxyResource := ProxyResource(instanceID, resourceInfo, connMgr)
		resources = append(resources, proxyResource)
	}
	
	// Register all resources with the server
	return server.RegisterResourcesFromInstance(instanceID, resources)
}

// UnregisterInstanceResources removes all resources from a disconnected instance
func UnregisterInstanceResources(server *StreamableServer, instanceID string) error {
	return server.UnregisterResourcesFromInstance(instanceID)
}

// ResourceSubscription handles resource subscriptions for proxied resources
type ResourceSubscription struct {
	instanceID   string
	resourceURI  string
	originalURI  string
	connMgr      *ConnectionManager
	updatesChan  chan ProxyResourceUpdate
	stopChan     chan struct{}
}

// ProxyResourceUpdate represents an update to a proxied resource
type ProxyResourceUpdate struct {
	URI       string      `json:"uri"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// NewResourceSubscription creates a new subscription to a proxied resource
func NewResourceSubscription(prefixedURI string, connMgr *ConnectionManager) (*ResourceSubscription, error) {
	instanceID, resourceURI, err := ExtractInstanceAndResource(prefixedURI)
	if err != nil {
		return nil, err
	}
	
	return &ResourceSubscription{
		instanceID:  instanceID,
		resourceURI: prefixedURI,
		originalURI: resourceURI,
		connMgr:     connMgr,
		updatesChan: make(chan ProxyResourceUpdate, 100),
		stopChan:    make(chan struct{}),
	}, nil
}

// Start begins monitoring the resource for updates
func (rs *ResourceSubscription) Start() error {
	// In a real implementation, this would:
	// 1. Subscribe to resource updates from the instance
	// 2. Forward updates to the updatesChan
	// 3. Handle reconnection if the instance connection drops
	
	// For now, we'll implement a simple polling mechanism
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-rs.stopChan:
				return
			case <-ticker.C:
				// Check if resource has updates
				// This is a placeholder - real implementation would
				// use proper subscription mechanism
				rs.checkForUpdates()
			}
		}
	}()
	
	return nil
}

// Stop stops monitoring the resource
func (rs *ResourceSubscription) Stop() {
	close(rs.stopChan)
	close(rs.updatesChan)
}

// Updates returns the channel for receiving updates
func (rs *ResourceSubscription) Updates() <-chan ProxyResourceUpdate {
	return rs.updatesChan
}

// checkForUpdates polls the resource for changes
func (rs *ResourceSubscription) checkForUpdates() {
	// Get the client for this instance
	connections := rs.connMgr.ListInstances()
	
	var activeClient *HubClient
	for _, conn := range connections {
		if conn.InstanceID == rs.instanceID && conn.State == StateActive && conn.Client != nil {
			activeClient = conn.Client
			break
		}
	}
	
	if activeClient == nil {
		return
	}
	
	// Read the resource
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := activeClient.ReadResource(ctx, rs.originalURI)
	if err != nil {
		return
	}
	
	// Send update
	select {
	case rs.updatesChan <- ProxyResourceUpdate{
		URI:       rs.resourceURI,
		Timestamp: time.Now(),
		Data:      result,
	}:
	default:
		// Channel full, drop update
	}
}