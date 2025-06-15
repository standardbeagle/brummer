package ports

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

// FindAvailablePort finds an available port starting from the given port
// If the port is in use, it tries random ports in the range [startPort, startPort+1000]
func FindAvailablePort(startPort int) (int, error) {
	// First try the requested port
	if isPortAvailable(startPort) {
		return startPort, nil
	}
	
	// If the requested port is not available, try random ports in a reasonable range
	rand.Seed(time.Now().UnixNano())
	maxAttempts := 50
	minPort := startPort
	maxPort := startPort + 1000
	
	// Ensure we stay within valid port range
	if maxPort > 65535 {
		maxPort = 65535
	}
	
	for attempts := 0; attempts < maxAttempts; attempts++ {
		// Generate a random port in the range
		randomPort := minPort + rand.Intn(maxPort-minPort+1)
		
		if isPortAvailable(randomPort) {
			return randomPort, nil
		}
	}
	
	return 0, fmt.Errorf("unable to find available port after %d attempts in range %d-%d", maxAttempts, minPort, maxPort)
}

// isPortAvailable checks if a port is available by attempting to listen on it
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}

// FindAvailablePortInRange finds an available port in the specified range
func FindAvailablePortInRange(minPort, maxPort int) (int, error) {
	if minPort > maxPort {
		return 0, fmt.Errorf("minPort (%d) must be <= maxPort (%d)", minPort, maxPort)
	}
	
	rand.Seed(time.Now().UnixNano())
	maxAttempts := 50
	
	for attempts := 0; attempts < maxAttempts; attempts++ {
		// Generate a random port in the range
		randomPort := minPort + rand.Intn(maxPort-minPort+1)
		
		if isPortAvailable(randomPort) {
			return randomPort, nil
		}
	}
	
	return 0, fmt.Errorf("unable to find available port after %d attempts in range %d-%d", maxAttempts, minPort, maxPort)
}