// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package scoring

import (
	"os"
	"sync"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

// Cache stores scored results keyed by job file path and the maximum
// modification time across experience files. If any experience file
// changes, the cache entry is invalidated.
type Cache struct {
	mu      sync.Mutex
	entries map[cacheKey][]ScoredExperience
}

type cacheKey struct {
	JobPath  string
	VaultMod int64 // max mtime (UnixNano) across experience files
}

// NewCache creates an empty score cache.
func NewCache() *Cache {
	return &Cache{
		entries: make(map[cacheKey][]ScoredExperience),
	}
}

// Get returns cached scoring results if available and still valid.
func (c *Cache) Get(job *vault.Job, experiences []*vault.Experience) ([]ScoredExperience, bool) {
	key := c.makeKey(job, experiences)

	c.mu.Lock()
	defer c.mu.Unlock()

	results, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	// Return a copy to prevent mutation.
	out := make([]ScoredExperience, len(results))
	copy(out, results)
	return out, true
}

// Put stores scoring results in the cache.
func (c *Cache) Put(job *vault.Job, experiences []*vault.Experience, results []ScoredExperience) {
	key := c.makeKey(job, experiences)

	// Store a copy.
	stored := make([]ScoredExperience, len(results))
	copy(stored, results)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = stored
}

// Invalidate removes cached entries for a specific job path.
func (c *Cache) Invalidate(jobPath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if key.JobPath == jobPath {
			delete(c.entries, key)
		}
	}
}

// Len returns the number of cached entries.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

func (c *Cache) makeKey(job *vault.Job, experiences []*vault.Experience) cacheKey {
	return cacheKey{
		JobPath:  job.FilePath,
		VaultMod: maxMtime(experiences),
	}
}

// maxMtime returns the maximum modification time across all experience file paths.
// Returns 0 if no files can be stat'd (e.g. in-memory test data).
func maxMtime(experiences []*vault.Experience) int64 {
	var max int64
	for _, exp := range experiences {
		if exp.FilePath == "" {
			continue
		}
		info, err := os.Stat(exp.FilePath)
		if err != nil {
			continue
		}
		mtime := info.ModTime().UnixNano()
		if mtime > max {
			max = mtime
		}
	}
	return max
}
