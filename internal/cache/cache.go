package cache

import (
	"fmt"
	"sync"
)

// CacheEntry represents a cached download entry
type CacheEntry struct {
	URL      string
	FilePath string
	FileID   string
}

// FileCache manages cached downloads with LRU eviction
type FileCache struct {
	maxSize int
	mu      sync.RWMutex
	// Key: userID:URL, Value: CacheEntry
	data map[string]CacheEntry
	// Track access order for LRU
	accessOrder []string
}

// NewFileCache creates a new file cache with the given max size
func NewFileCache(maxSize int) *FileCache {
	return &FileCache{
		maxSize:     maxSize,
		data:        make(map[string]CacheEntry),
		accessOrder: make([]string, 0, maxSize),
	}
}

// Get retrieves a cached file ID for a given user and URL
func (fc *FileCache) Get(userID int64, url string) (fileID string, found bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	key := cacheKey(userID, url)
	entry, exists := fc.data[key]
	if exists {
		return entry.FileID, true
	}
	return "", false
}

// Add adds or updates a cache entry for a given user and URL
func (fc *FileCache) Add(userID int64, url, filePath, fileID string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	key := cacheKey(userID, url)

	// If entry already exists, remove it from access order
	if _, exists := fc.data[key]; exists {
		// Remove from access order
		for i, k := range fc.accessOrder {
			if k == key {
				fc.accessOrder = append(fc.accessOrder[:i], fc.accessOrder[i+1:]...)
				break
			}
		}
	}

	// Add or update the entry
	fc.data[key] = CacheEntry{
		URL:      url,
		FilePath: filePath,
		FileID:   fileID,
	}
	fc.accessOrder = append(fc.accessOrder, key)

	// Evict oldest entry if we exceed max size
	if len(fc.data) > fc.maxSize && len(fc.accessOrder) > 0 {
		oldestKey := fc.accessOrder[0]
		fc.accessOrder = fc.accessOrder[1:]
		delete(fc.data, oldestKey)
	}
}

// Remove deletes a cache entry for a given user and URL
func (fc *FileCache) Remove(userID int64, url string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	key := cacheKey(userID, url)

	if _, exists := fc.data[key]; exists {
		delete(fc.data, key)

		// Remove from access order
		for i, k := range fc.accessOrder {
			if k == key {
				fc.accessOrder = append(fc.accessOrder[:i], fc.accessOrder[i+1:]...)
				break
			}
		}
	}
}

// LoadFromHistory populates the cache with historical entries
func (fc *FileCache) LoadFromHistory(entries []CacheEntry, userIDs []int64) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Clear existing data
	fc.data = make(map[string]CacheEntry)
	fc.accessOrder = make([]string, 0, fc.maxSize)

	// Load entries from history, keeping the most recent ones (up to maxSize)
	// The entries are provided in ascending order by created_at, so we reverse to get newest first
	for i := len(entries) - 1; i >= 0 && len(fc.data) < fc.maxSize; i-- {
		key := cacheKey(userIDs[i], entries[i].URL)
		if _, exists := fc.data[key]; !exists {
			fc.data[key] = entries[i]
			fc.accessOrder = append(fc.accessOrder, key)
		}
	}
}

// cacheKey creates a unique cache key from userID and URL
func cacheKey(userID int64, url string) string {
	return fmt.Sprintf("%d:%s", userID, url)
}
