package state

import (
	"sync"
	"time"
)

// Cache provides a thread-safe TTL cache for storing application data
type Cache struct {
	entries    map[string]*CacheEntry
	mu         sync.RWMutex
	maxSize    int
	ttl        time.Duration
	stats      *CacheStats
	cleanup    *CacheCleanup
	stopChan   chan struct{}
}

// CacheEntry represents a cached value with metadata
type CacheEntry struct {
	Key        string                 `json:"key"`
	Value      interface{}            `json:"value"`
	ExpiresAt  time.Time              `json:"expiresAt"`
	CreatedAt  time.Time              `json:"createdAt"`
	AccessedAt time.Time              `json:"accessedAt"`
	Size       int64                  `json:"size"`
	Version    uint64                 `json:"version"`
	Hits       int64                  `json:"hits"`
	Misses     int64                  `json:"misses"`
	TTL        time.Duration          `json:"ttl"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CacheStats provides statistics about cache performance
type CacheStats struct {
	TotalEntries     int64         `json:"totalEntries"`
	ActiveEntries    int64         `json:"activeEntries"`
	Hits              int64         `json:"hits"`
	Misses            int64         `json:"misses"`
	HitRate           float64       `json:"hitRate"`
	Evictions         int64         `json:"evictions"`
	TotalSize         int64         `json:"totalSize"`
	AverageEntrySize  float64       `json:"averageEntrySize"`
	OldestEntry       time.Time     `json:"oldestEntry"`
	NewestEntry       time.Time     `json:"newestEntry"`
	LastCleanup       time.Time     `json:"lastCleanup"`
	CleanupCount      int64         `json:"cleanupCount"`
}

// CacheCleanup manages automatic cleanup of expired entries
type CacheCleanup struct {
	interval   time.Duration
	stopChan  chan struct{}
	enabled   bool
	ticker    *time.Ticker
}

// CacheFilter provides filtering capabilities for cache entries
type CacheFilter struct {
	Prefix     string        `json:"prefix,omitempty"`
	Suffix     string        `json:"suffix,omitempty"`
	Expired    *bool         `json:"expired,omitempty"`
	MinSize    *int64        `json:"minSize,omitempty"`
	MaxSize    *int64        `json:"maxSize,omitempty"`
	KeyPattern string        `json:"keyPattern,omitempty"`
}

// NewCache creates a new cache with specified size and TTL
func NewCache(maxSize int, ttl time.Duration) *Cache {
	cache := &Cache{
	entries:  make(map[string]*CacheEntry),
		maxSize:  maxSize,
		ttl:      ttl,
		stats:    NewCacheStats(),
		stopChan: make(chan struct{}),
	}

	// Start cleanup routine
	cache.cleanup = NewCacheCleanup(ttl / 4) // Cleanup every 1/4 of TTL
	cache.startCleanup()

	return cache
}

// NewCacheStats creates new cache statistics
func NewCacheStats() *CacheStats {
	return &CacheStats{
		HitRate: 0,
	}
}

// NewCacheCleanup creates a new cache cleanup routine
func NewCacheCleanup(interval time.Duration) *CacheCleanup {
	return &CacheCleanup{
		interval:  interval,
		stopChan:  make(chan struct{}),
		enabled:  true,
	}
}

// startCleanup initializes cleanup but does not start automatic goroutine
// Cleanup must be called manually to maintain Bubble Tea MVU pattern
func (c *Cache) startCleanup() {
	// Cleanup is now manual - call cleanupExpired() when needed
}

// cleanupExpired removes expired entries from the cache
func (c *Cache) cleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	var expiredKeys []string

	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.entries, key)
		c.stats.Evictions++
	}

	if len(expiredKeys) > 0 {
		c.stats.LastCleanup = now
		c.stats.CleanupCount++
		c.updateStats()
	}
}

// Stop stops the cache cleanup routine
func (c *Cache) Stop() {
	close(c.stopChan)
	if c.cleanup != nil {
		close(c.cleanup.stopChan)
	}
}

// Set stores a value in the cache with the default TTL
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a specific TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// Check if we need to make space
	if len(c.entries) >= c.maxSize {
		c.evictLRU()
	}

	// Create new entry
	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		ExpiresAt:  now.Add(ttl),
		CreatedAt:  now,
		AccessedAt: now,
		Size:       c.estimateSize(value),
		Hits:       0,
		Misses:     0,
		TTL:        ttl,
		Metadata:   make(map[string]interface{}),
	}

	// Update stats if replacing existing entry
	if oldEntry, exists := c.entries[key]; exists {
		entry.Hits = oldEntry.Hits
		entry.Misses = oldEntry.Misses
	} else {
		c.stats.Misses++
	}

	c.entries[key] = entry
	c.updateStats()
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		c.updateStats()
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.stats.Misses++
		c.updateStats()
		return nil, false
	}

	// Update access time and hit count
	entry.AccessedAt = time.Now()
	entry.Hits++
	c.stats.Hits++
	c.updateStats()

	return entry.Value, true
}

// GetWithMetadata retrieves a value and its metadata from the cache
func (c *Cache) GetWithMetadata(key string) (interface{}, map[string]interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		c.updateStats()
		return nil, nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.stats.Misses++
		c.updateStats()
		return nil, nil, false
	}

	// Update access time and hit count
	entry.AccessedAt = time.Now()
	entry.Hits++
	c.stats.Hits++
	c.updateStats()

	// Prepare metadata
	metadata := map[string]interface{}{
		"expiresAt": entry.ExpiresAt,
		"createdAt": entry.CreatedAt,
		"accessedAt": entry.AccessedAt,
		"size":       entry.Size,
		"hits":       entry.Hits,
		"misses":     entry.Misses,
		"ttl":        entry.TTL,
	}

	for k, v := range entry.Metadata {
		metadata[k] = v
	}

	return entry.Value, metadata, true
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.entries[key]
	if !exists {
		return false
	}

	delete(c.entries, key)
	c.updateStats()

	return true
}

// Exists checks if a key exists in the cache (regardless of expiration)
func (c *Cache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.entries[key]
	return exists
}

// ExistsFresh checks if a key exists and is not expired
func (c *Cache) ExistsFresh(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return false
	}

	return !time.Now().After(entry.ExpiresAt)
}

// GetTTL returns the remaining TTL for a key
func (c *Cache) GetTTL(key string) (time.Duration, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return 0, false
	}

	remaining := entry.ExpiresAt.Sub(time.Now())
	if remaining <= 0 {
		return 0, false
	}

	return remaining, true
}

// ExtendTTL extends the TTL for an existing key
func (c *Cache) ExtendTTL(key string, additionalTTL time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		return false
	}

	entry.ExpiresAt = entry.ExpiresAt.Add(additionalTTL)
	entry.TTL = entry.TTL + additionalTTL
	entry.AccessedAt = time.Now()

	return true
}

// SetMetadata sets metadata for a cache entry
func (c *Cache) SetMetadata(key string, metadata map[string]interface{}) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		return false
	}

	if entry.Metadata == nil {
		entry.Metadata = make(map[string]interface{})
	}

	for k, v := range metadata {
		entry.Metadata[k] = v
	}

	entry.AccessedAt = time.Now()

	return true
}

// GetMetadata gets metadata for a cache entry
func (c *Cache) GetMetadata(key string) (map[string]interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	return entry.Metadata, true
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.updateStats()
}

// Keys returns all keys in the cache
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.entries))
	for key := range c.entries {
		keys = append(keys, key)
	}

	return keys
}

// GetFilteredKeys returns keys filtered by the provided filter
func (c *Cache) GetFilteredKeys(filter *CacheFilter) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var keys []string
	for key := range c.entries {
		if c.matchesFilter(key, filter) {
			keys = append(keys, key)
		}
	}

	return keys
}

// matchesFilter checks if a key matches the provided filter
func (c *Cache) matchesFilter(key string, filter *CacheFilter) bool {
	// Check prefix filter
	if filter.Prefix != "" && !c.startsWith(key, filter.Prefix) {
		return false
	}

	// Check suffix filter
	if filter.Suffix != "" && !c.endsWith(key, filter.Suffix) {
		return false
	}

	// Check expiration filter
	if filter.Expired != nil {
		entry, exists := c.entries[key]
		if !exists {
			return false
		}
		isExpired := time.Now().After(entry.ExpiresAt)
		if isExpired != *filter.Expired {
			return false
		}
	}

	// Check size filters
	entry, exists := c.entries[key]
	if !exists {
		return false
	}

	if filter.MinSize != nil && entry.Size < *filter.MinSize {
		return false
	}
	if filter.MaxSize != nil && entry.Size > *filter.MaxSize {
		return false
	}

	// Check key pattern
	if filter.KeyPattern != "" && !c.matchesPattern(key, filter.KeyPattern) {
		return false
	}

	return true
}

// startsWith checks if a string starts with a prefix
func (c *Cache) startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// endsWith checks if a string ends with a suffix
func (c *Cache) endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// matchesPattern checks if a string matches a simple pattern
func (c *Cache) matchesPattern(s, pattern string) bool {
	// Simple wildcard matching for now
	if pattern == "*" {
		return true
	}

	// Exact match
	if s == pattern {
		return true
	}

	// Simple wildcard support
	if len(pattern) > 1 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		substr := pattern[1 : len(pattern)-1]
		return c.contains(s, substr)
	}

	return false
}

// contains checks if a string contains a substring
func (c *Cache) contains(s, substr string) bool {
	return len(s) >= len(substr) && c.findSubstring(s, substr) >= 0
}

// findSubstring finds the index of a substring
func (c *Cache) findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetEntries returns all cache entries that match the filter
func (c *Cache) GetEntries(filter *CacheFilter) map[string]*CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*CacheEntry)
	for key, entry := range c.entries {
		// Skip expired entries unless specifically requested
		if filter.Expired == nil || *filter.Expired {
			if time.Now().After(entry.ExpiresAt) {
				continue
			}
		}

		if c.matchesFilter(key, filter) {
			result[key] = c.copyEntry(entry)
		}
	}

	return result
}

// Pop retrieves and removes a value from the cache
func (c *Cache) Pop(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		c.updateStats()
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, key)
		c.stats.Misses++
		c.updateStats()
		return nil, false
	}

	// Remove and return
	delete(c.entries, key)
	c.stats.Hits++
	c.updateStats()

	return entry.Value, true
}

// evictLRU removes the least recently used entry
func (c *Cache) evictLRU() {
	var oldestKey string
	var oldestAccess time.Time

	for key, entry := range c.entries {
		if oldestAccess.IsZero() || entry.AccessedAt.Before(oldestAccess) {
			oldestKey = key
			oldestAccess = entry.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.stats.Evictions++
	}
}

// evictLRUCount removes multiple least recently used entries
func (c *Cache) evictLRUCount(count int) {
	for i := 0; i < count && len(c.entries) > 0; i++ {
		c.evictLRU()
	}
}

// Resize changes the maximum size of the cache
func (c *Cache) Resize(newMaxSize int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxSize = newMaxSize

	// Remove excess entries if necessary
	for len(c.entries) > c.maxSize {
		c.evictLRU()
	}
}

// updateStats updates cache statistics
func (c *Cache) updateStats() {
	c.stats.TotalEntries = int64(len(c.entries))
	c.stats.ActiveEntries = int64(len(c.entries))
	c.stats.HitRate = c.calculateHitRate()
	c.stats.TotalSize = c.calculateTotalSize()
	c.stats.AverageEntrySize = c.calculateAverageSize()
	c.stats.OldestEntry = c.getOldestEntry()
	c.stats.NewestEntry = c.getNewestEntry()
}

// calculateHitRate calculates the hit rate
func (c *Cache) calculateHitRate() float64 {
	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total)
}

// calculateTotalSize calculates the total size of all entries
func (c *Cache) calculateTotalSize() int64 {
	var total int64
	for _, entry := range c.entries {
		total += entry.Size
	}
	return total
}

// calculateAverageSize calculates the average entry size
func (c *Cache) calculateAverageSize() float64 {
	if len(c.entries) == 0 {
		return 0
	}
	return float64(c.calculateTotalSize()) / float64(len(c.entries))
}

// getOldestEntry finds the oldest entry in the cache
func (c *Cache) getOldestEntry() time.Time {
	var oldest time.Time
	for _, entry := range c.entries {
		if oldest.IsZero() || entry.CreatedAt.Before(oldest) {
			oldest = entry.CreatedAt
		}
	}
	return oldest
}

// getNewestEntry finds the newest entry in the cache
func (c *Cache) getNewestEntry() time.Time {
	var newest time.Time
	for _, entry := range c.entries {
		if newest.IsZero() || entry.CreatedAt.After(newest) {
			newest = entry.CreatedAt
		}
	}
	return newest
}

// estimateSize estimates the size of a value
func (c *Cache) estimateSize(value interface{}) int64 {
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case bool:
		return 1
	case int, int8, int16, int32:
		return 8
	case int64:
		return 8
	case uint, uint8, uint16, uint32:
		return 8
	case uint64:
		return 8
	case float32, float64:
		return 8
	case time.Time:
		return 24
	case map[string]interface{}:
		size := int64(0)
		for k, v := range v {
			size += int64(len(k)) + c.estimateSize(v)
		}
		return size
	case []interface{}:
		size := int64(0)
		for _, v := range v {
			size += c.estimateSize(v)
		}
		return size
	default:
		// Use reflection to get a rough estimate
		return 100
	}
}

// copyEntry creates a deep copy of a cache entry
func (c *Cache) copyEntry(original *CacheEntry) *CacheEntry {
	copy := &CacheEntry{
		Key:        original.Key,
		Value:      original.Value,
		ExpiresAt:  original.ExpiresAt,
		CreatedAt:  original.CreatedAt,
		AccessedAt: original.AccessedAt,
		Size:       original.Size,
		Version:    original.Version,
		Hits:       original.Hits,
		Misses:     original.Misses,
		TTL:        original.TTL,
		Metadata:   make(map[string]interface{}),
	}

	for k, v := range original.Metadata {
		copy.Metadata[k] = v
	}

	return copy
}

// GetStats returns cache statistics
func (c *Cache) GetStats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return &CacheStats{
		TotalEntries:     c.stats.TotalEntries,
		ActiveEntries:    c.stats.ActiveEntries,
		Hits:              c.stats.Hits,
		Misses:            c.stats.Misses,
		HitRate:           c.stats.HitRate,
		Evictions:         c.stats.Evictions,
		TotalSize:         c.stats.TotalSize,
		AverageEntrySize:  c.stats.AverageEntrySize,
		OldestEntry:       c.stats.OldestEntry,
		NewestEntry:       c.stats.NewestEntry,
		LastCleanup:       c.stats.LastCleanup,
		CleanupCount:      c.stats.CleanupCount,
	}
}

// ResetStats resets cache statistics
func (c *Cache) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats = NewCacheStats()
}

// Export exports cache data for persistence
func (c *Cache) Export() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"entries": c.entries,
		"stats":   c.stats,
		"config": map[string]interface{}{
			"maxSize": c.maxSize,
			"ttl":     c.ttl,
		},
		"exportedAt": time.Now(),
	}
}

// Import imports cache data from exported data
func (c *Cache) Import(data map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear current state
	c.Clear()

	// Import entries
	if entriesData, ok := data["entries"].(map[string]*CacheEntry); ok {
		for _, entry := range entriesData {
			c.entries[entry.Key] = entry
		}
	}

	// Import config
	if configData, ok := data["config"].(map[string]interface{}); ok {
		if maxSize, ok := configData["maxSize"].(int); ok {
			c.maxSize = maxSize
		}
		if ttl, ok := configData["ttl"].(time.Duration); ok {
			c.ttl = ttl
		}
	}

	// Import stats
	if statsData, ok := data["stats"].(*CacheStats); ok {
		c.stats = statsData
	}

	// Restart cleanup
	c.Stop()
	c.cleanup = NewCacheCleanup(c.ttl / 4)
	c.startCleanup()

	return nil
}

// Warmup pre-populates the cache with the provided data
func (c *Cache) Warmup(data map[string]interface{}) {
	for key, value := range data {
		c.Set(key, value)
	}
}

// Size returns the current number of entries in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// IsEmpty returns true if the cache is empty
func (c *Cache) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries) == 0
}

// IsFull returns true if the cache is at capacity
func (c *Cache) IsFull() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries) >= c.maxSize
}

// GetMaxSize returns the maximum cache size
func (c *Cache) GetMaxSize() int {
	return c.maxSize
}


// MemoryUsage returns estimated memory usage in bytes
func (c *Cache) MemoryUsage() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Basic estimation: each entry plus overhead
	overheadPerEntry := int64(200) // Rough estimate of struct overhead
	return c.stats.TotalSize + int64(len(c.entries))*overheadPerEntry
}