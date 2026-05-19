package cache

import (
	"log"
	"os"
	"sync"
	"time"
)

type CacheEntry struct {
	URL       string
	FilePath  string
	FileID    string // Telegram file_id for quick resending
	CreatedAt time.Time
}

type FileCache struct {
	mu         sync.RWMutex
	maxPerUser int
	entries    map[int64][]CacheEntry // UserID -> list of entries (up to 3)
}

func NewFileCache(maxPerUser int) *FileCache {
	return &FileCache{
		maxPerUser: maxPerUser,
		entries:    make(map[int64][]CacheEntry),
	}
}

// Add appends a new file entry to the user's cache.
// If the user exceeds maxPerUser, the oldest cached file is deleted.
func (c *FileCache) Add(userID int64, url string, filePath string, fileID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	userList := c.entries[userID]

	// Check if this URL is already in the user's cache to avoid duplication
	for i, entry := range userList {
		if entry.URL == url {
			// Update the entry with new fileID and timestamp, move to the end (LRU-like)
			userList[i].FileID = fileID
			userList[i].CreatedAt = time.Now()
			// Re-slice to make it the most recent (end of slice)
			item := userList[i]
			userList = append(userList[:i], userList[i+1:]...)
			userList = append(userList, item)
			c.entries[userID] = userList
			return
		}
	}

	newEntry := CacheEntry{
		URL:       url,
		FilePath:  filePath,
		FileID:    fileID,
		CreatedAt: time.Now(),
	}

	userList = append(userList, newEntry)

	// If user has more than limit, evict the oldest
	if len(userList) > c.maxPerUser {
		evicted := userList[0]
		userList = userList[1:]

		// Clean up file if it's not referenced by any other entry (either for this user or any other)
		c.deleteIfUnreferenced(evicted.FilePath, userID, evicted.URL)
	}

	c.entries[userID] = userList
	log.Printf("Cached file for user %d: %s (total cached: %d)", userID, filePath, len(userList))
}

// Get checks if a URL is cached for a specific user.
// Returns the fileID and true if it exists.
func (c *FileCache) Get(userID int64, url string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	userList, exists := c.entries[userID]
	if !exists {
		return "", false
	}

	for _, entry := range userList {
		if entry.URL == url {
			// Double check if file actually exists on disk (if FilePath is populated)
			if entry.FilePath != "" {
				if _, err := os.Stat(entry.FilePath); os.IsNotExist(err) {
					return "", false
				}
			}
			return entry.FileID, true
		}
	}

	return "", false
}

// deleteIfUnreferenced deletes a file only if it is not in use by any other cache entry.
func (c *FileCache) deleteIfUnreferenced(filePath string, excludeUserID int64, excludeURL string) {
	if filePath == "" {
		return
	}

	referenced := false
	for uID, list := range c.entries {
		for _, entry := range list {
			// Skip the one we are evicting
			if uID == excludeUserID && entry.URL == excludeURL {
				continue
			}
			if entry.FilePath == filePath {
				referenced = true
				break
			}
		}
		if referenced {
			break
		}
	}

	if !referenced {
		log.Printf("Cache eviction: deleting physical file %s", filePath)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			log.Printf("Failed to delete evicted file %s: %v", filePath, err)
		}
	}
}

// LoadFromHistory populates the cache on startup using SQLite database records
func (c *FileCache) LoadFromHistory(history []CacheEntry, userIDs []int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Direct bulk loading mapping each user to their 3 most recent files
	for i, uID := range userIDs {
		c.entries[uID] = append(c.entries[uID], history[i])
	}
	log.Printf("Pre-populated file cache for %d users from database history", len(c.entries))
}

// Remove deletes a cache entry for a user if it matches the URL.
func (c *FileCache) Remove(userID int64, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	userList, exists := c.entries[userID]
	if !exists {
		return
	}

	for i, entry := range userList {
		if entry.URL == url {
			// Remove from slice
			c.entries[userID] = append(userList[:i], userList[i+1:]...)
			return
		}
	}
}
