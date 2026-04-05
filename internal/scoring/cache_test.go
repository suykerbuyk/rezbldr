// Copyright (c) 2026 John Suykerbuyk and SykeTech LTD
// SPDX-License-Identifier: MIT OR Apache-2.0

package scoring

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/suykerbuyk/rezbldr/internal/vault"
)

func TestCache_MissAndHit(t *testing.T) {
	c := NewCache()
	job := bigcoJob()
	exps := testExperiences()

	// Miss on empty cache.
	_, ok := c.Get(job, exps)
	if ok {
		t.Fatal("expected cache miss on empty cache")
	}

	// Put results.
	results := []ScoredExperience{
		{File: "test.md", Score: 5.0, NormalizedScore: 100},
	}
	c.Put(job, exps, results)

	// Hit.
	cached, ok := c.Get(job, exps)
	if !ok {
		t.Fatal("expected cache hit after Put")
	}
	if len(cached) != 1 || cached[0].Score != 5.0 {
		t.Errorf("cached result mismatch: %v", cached)
	}
}

func TestCache_ReturnsCopy(t *testing.T) {
	c := NewCache()
	job := &vault.Job{Title: "T", Company: "C", FilePath: "/job.md"}

	results := []ScoredExperience{{File: "a.md", Score: 1.0}}
	c.Put(job, nil, results)

	// Mutate the returned copy — should not affect cache.
	cached, _ := c.Get(job, nil)
	cached[0].Score = 999.0

	cached2, _ := c.Get(job, nil)
	if cached2[0].Score != 1.0 {
		t.Error("cache should return independent copies")
	}
}

func TestCache_InvalidateByJobPath(t *testing.T) {
	c := NewCache()
	job := &vault.Job{Title: "T", Company: "C", FilePath: "/job.md"}

	c.Put(job, nil, []ScoredExperience{{File: "a.md"}})
	if c.Len() != 1 {
		t.Fatal("expected 1 entry")
	}

	c.Invalidate("/job.md")
	if c.Len() != 0 {
		t.Error("expected 0 entries after invalidation")
	}

	_, ok := c.Get(job, nil)
	if ok {
		t.Error("expected miss after invalidation")
	}
}

func TestCache_InvalidationByMtime(t *testing.T) {
	// Create a temp file to use as an experience file path.
	dir := t.TempDir()
	expPath := filepath.Join(dir, "exp.md")
	if err := os.WriteFile(expPath, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	job := &vault.Job{Title: "T", Company: "C", FilePath: "/job.md"}
	exps := []*vault.Experience{{
		Role: "R", Company: "C", Visibility: "resume", FilePath: expPath,
	}}

	c := NewCache()
	c.Put(job, exps, []ScoredExperience{{File: "exp.md", Score: 1.0}})

	// Should hit.
	_, ok := c.Get(job, exps)
	if !ok {
		t.Fatal("expected cache hit")
	}

	// Touch the file to change mtime.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(expPath, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should miss because mtime changed.
	_, ok = c.Get(job, exps)
	if ok {
		t.Error("expected cache miss after file modification")
	}
}

func TestCache_Len(t *testing.T) {
	c := NewCache()
	if c.Len() != 0 {
		t.Error("new cache should be empty")
	}

	c.Put(&vault.Job{FilePath: "/a.md"}, nil, nil)
	c.Put(&vault.Job{FilePath: "/b.md"}, nil, nil)
	if c.Len() != 2 {
		t.Errorf("expected 2 entries, got %d", c.Len())
	}
}
