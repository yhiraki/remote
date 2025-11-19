package host

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Get resolves the remote hostname, utilizing a cache file to minimize command execution.
func Get(
	cmd string, cacheFile string, cacheExpireMinutes int, isVerbose bool,
) (string, error) {
	timeBeforeCacheExpires := time.Duration(cacheExpireMinutes) * time.Minute

	if isVerbose {
		log.Printf("[DEBUG] Checking cache file: %s", cacheFile)
	}

	// First, try to read from a valid cache
	cacheFileState, err := os.Stat(cacheFile)
	if err == nil { // Cache file exists
		if isVerbose {
			log.Printf("[DEBUG] Cache file found. Checking expiration...")
		}
		isCacheExpired := cacheFileState.ModTime().Add(timeBeforeCacheExpires).Before(time.Now())
		if !isCacheExpired {
			if isVerbose {
				log.Printf("[DEBUG] Cache is not expired. Reading content...")
			}
			content, err := os.ReadFile(cacheFile)
			if err == nil {
				host := strings.TrimSpace(string(content))
				if host != "" {
					if isVerbose {
						log.Printf("[DEBUG] Cache hit. Using hostname from cache: \"%s\"", host)
					}
					return host, nil
				}
				// Cache is empty
				if isVerbose {
					log.Printf("[DEBUG] Cache is empty. Will execute command.")
				}
			}
		}
		// Cache is expired
		if isVerbose {
			log.Printf("[DEBUG] Cache is expired. Will execute command.")
		}
	} else { // Cache file does not exist
		if isVerbose {
			log.Printf("[DEBUG] Cache file does not exist. Will execute command.")
		}
	}

	// If cache is non-existent, expired, or empty, fetch the hostname by running the command
	if isVerbose {
		log.Printf("[DEBUG] Attempting to get hostname from command: \"%s\"", cmd)
	}
	shcmd := strings.Split(cmd, "\n")[0]
	parts := strings.Fields(shcmd)
	if len(parts) == 0 {
		return "", errors.New("Hostname command is empty")
	}
	cmdName := parts[0]
	cmdArgs := parts[1:]
	out, err := exec.Command(cmdName, cmdArgs...).Output()
	if err != nil {
		// If command fails, remove the potentially empty/stale cache file to force refetch next time.
		if isVerbose {
			log.Printf("[ERROR] Command execution failed. Error: %v", err)
			if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
				log.Printf("[ERROR] Command Stderr: %s", strings.TrimSpace(string(exitErr.Stderr)))
			}
			log.Printf("[DEBUG] Removing stale cache file: %s", cacheFile)
		}
		os.Remove(cacheFile)
		return "", errors.New("Could not get hostname from command")
	}
	if isVerbose {
		log.Printf("[DEBUG] Command executed successfully.")
		log.Printf("[DEBUG] Raw command output: \"%s\"", strings.TrimSpace(string(out)))
	}

	hostname := strings.TrimSpace(string(out))
	if hostname == "" {
		if isVerbose {
			log.Printf("[ERROR] Command returned empty output.")
			log.Printf("[DEBUG] Removing stale cache file: %s", cacheFile)
		}
		os.Remove(cacheFile)
		return "", errors.New("Hostname command returned an empty string")
	}

	if isVerbose {
		log.Printf("[DEBUG] Trimmed hostname: \"%s\"", hostname)
		log.Printf("[DEBUG] Writing new hostname to cache file: %s", cacheFile)
	}
	// Write the newly fetched hostname to the cache file
	err = os.WriteFile(cacheFile, []byte(hostname), 0644)
	if err != nil {
		if isVerbose {
			log.Printf("[ERROR] Failed to write to cache file. Error: %v", err)
		}
		return "", fmt.Errorf("Could not write to hostname cachefile: %w", err)
	}

	if isVerbose {
		log.Printf("[DEBUG] Successfully wrote to cache.")
	}
	return hostname, nil
}
