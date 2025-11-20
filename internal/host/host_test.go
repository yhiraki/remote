package host

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remote_host_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cacheFile := filepath.Join(tmpDir, "hostname_cache")

	t.Run("cache miss and command execution", func(t *testing.T) {
		// Use a simple echo command that is cross-platform enough (usually available in basic shells)
		// or just "echo" if we assume unix-like environment as per original code structure (ssh/rsync usage)
		cmd := "echo example.com"
		
		host, err := Get(cmd, cacheFile, 60, false)
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if host != "example.com" {
			t.Errorf("Get() host = %v, want %v", host, "example.com")
		}

		// Verify cache file created
		content, err := os.ReadFile(cacheFile)
		if err != nil {
			t.Errorf("Cache file not created: %v", err)
		}
		if string(content) != "example.com" {
			t.Errorf("Cache content = %v, want %v", string(content), "example.com")
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		// Pre-populate cache with a different value to verify it's used
		cachedHost := "cached.example.com"
		if err := os.WriteFile(cacheFile, []byte(cachedHost), 0644); err != nil {
			t.Fatal(err)
		}

		// Set modification time to now to ensure it's valid
		now := time.Now()
		if err := os.Chtimes(cacheFile, now, now); err != nil {
			t.Fatal(err)
		}

		// Command that would return something else if executed
		cmd := "echo new.example.com"

		host, err := Get(cmd, cacheFile, 60, false)
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if host != cachedHost {
			t.Errorf("Get() host = %v, want %v (from cache)", host, cachedHost)
		}
	})

	t.Run("cache expired", func(t *testing.T) {
		// Pre-populate cache
		cachedHost := "expired.example.com"
		if err := os.WriteFile(cacheFile, []byte(cachedHost), 0644); err != nil {
			t.Fatal(err)
		}

		// Set modification time to past
		past := time.Now().Add(-61 * time.Minute)
		if err := os.Chtimes(cacheFile, past, past); err != nil {
			t.Fatal(err)
		}

		cmd := "echo new.example.com"
		host, err := Get(cmd, cacheFile, 60, false)
		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if host != "new.example.com" {
			t.Errorf("Get() host = %v, want %v (refreshed)", host, "new.example.com")
		}
	})
}

