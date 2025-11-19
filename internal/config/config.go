package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	Hostname           string   `json:"hostname"`
	HostnameCommand    string   `json:"hostnameCommand"`
	ExcludeFiles       []string `json:"excludeFiles"`
	ConfigDir          string   `json:"configDir"`
	CacheDir           string   `json:"cacheDir"`
	CacheExpireMinutes int      `json:"cacheExpireMinutes"`
	StartupWaitSeconds int      `json:"startupWaitSeconds"`
}

func New() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &Config{
		Hostname:           "",
		HostnameCommand:    "",
		ExcludeFiles:       []string{},
		ConfigDir:          filepath.Join(home, ".config", "remote"),
		CacheDir:           filepath.Join(home, ".cache", "remote"),
		CacheExpireMinutes: 12 * 60,
		StartupWaitSeconds: 20,
	}, nil
}

// Load finds and loads configuration from file
func (c *Config) Load(fileName string) error {
	configFile, err := findConfigFile(fileName)
	if err == nil {
		// project local config found
		// Adjust ConfigDir and CacheDir based on found config file location
		c.ConfigDir = filepath.Dir(configFile)
		c.CacheDir = filepath.Join(c.ConfigDir, ".remote")
	} else {
		// user global config
		configFile = filepath.Join(c.ConfigDir, fileName)
	}

	return parseConfigJson(configFile, c)
}

func findConfigFile(name string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	currentDir := cwd
	for {
		path := filepath.Join(currentDir, name)
		if s, err := os.Stat(path); err == nil && !s.IsDir() {
			return path, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir { // If reached root directory
			break
		}
		currentDir = parentDir
	}
	return "", errors.New("Config path not found.")
}

func parseConfigJson(fileName string, config *Config) error {
	fp, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer fp.Close()

	if err := json.NewDecoder(fp).Decode(config); err != nil {
		return err
	}
	return nil
}
