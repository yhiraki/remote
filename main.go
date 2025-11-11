package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

var (
	home string
	cwd  string
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

func NewConfig() Config {
	return Config{
		"",
		"",
		[]string{},
		filepath.Join(home, ".config", "remote"),
		filepath.Join(home, ".cache", "remote"),
		12 * 60,
		20,
	}
}

// Find nearest config file path
func findConfigFile(name string) (string, error) {
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

	if err := json.NewDecoder(fp).Decode(&config); err != nil {
		return err
	}
	return nil
}

func init() {
	var err error

	home, err = os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
}

// get remote hostname and cache
func getRemoteHostname(
	cmd string, cacheFile string, timeBeforCacheExpies time.Duration, isVerbose bool,
) (string, error) {
	if isVerbose {
		log.Printf("[DEBUG] Checking cache file: %s", cacheFile)
	}

	// First, try to read from a valid cache
	cacheFileState, err := os.Stat(cacheFile)
	if err == nil { // Cache file exists
		if isVerbose {
			log.Printf("[DEBUG] Cache file found. Checking expiration...")
		}
		isCacheExpired := cacheFileState.ModTime().Add(timeBeforCacheExpies).Before(time.Now())
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

type stringSlice []string

func (i *stringSlice) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func _main() error {
	config := NewConfig()

	configName := ".remoterc.json"
	configFile, err := findConfigFile(configName)
	if err == nil {
		// project local cacheDir
		config.ConfigDir = filepath.Join(configFile, "..")
		config.CacheDir = filepath.Join(config.ConfigDir, ".remote")
	} else {
		// user global cacheDir and configDir
		configFile = filepath.Join(config.ConfigDir, configName)
	}
	if err := parseConfigJson(configFile, &config); err != nil {
		log.Printf("%s could not parsed.", configFile)
		return err
	}

	// create directories
	for _, d := range []string{config.CacheDir, config.ConfigDir} {
		if _, err := os.Stat(d); err != nil {
			if err = os.MkdirAll(d, 0o705); err != nil {
				return err
			}
		}
	}

	// command line parsing
	var envVars stringSlice
	flag.Var(&envVars, "e", "set environment variable (e.g. -e KEY=VALUE)")
	flag.Var(&envVars, "env", "set environment variable (e.g. --env KEY=VALUE)")
	isDryRun := flag.Bool("dry-run", false, "dry run")
	isVerbose := flag.Bool("verbose", false, "enable verbose logging")
	showVersion := flag.Bool("version", false, "print version information")
	isBackground := flag.Bool("background", false, "run tunnel in background")
	flag.Parse()

	if *showVersion {
		if info, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(info.Main.Version)
		} else {
			fmt.Println("version not found")
		}
		return nil
	}

	// get hostname
	host := config.Hostname
	if config.HostnameCommand != "" {
		timeBeforCacheExpies := time.Duration(config.CacheExpireMinutes) * time.Minute
		host, err = getRemoteHostname(
			config.HostnameCommand,
			filepath.Join(config.CacheDir, "hostname"),
			timeBeforCacheExpies,
			*isVerbose)
		if err != nil {
			return err
		}
	}

	// get relative current path
	cwdRel, err := filepath.Rel(home, cwd)
	if err != nil {
		return err
	}

	// build command args
	cmdName, cmdArg, err := func(host string, args []string) (string, []string, error) {
		subCmd := "sh"
		subCmdArgs := []string{}
		if len(args) > 0 {
			subCmd = args[0]
			subCmdArgs = args[1:]
		}

		// ip
		if subCmd == "ip" {
			fmt.Println(host)
			return "", nil, nil
		}

		// ssh

		if subCmd == "" || subCmd == "sh" {
			shCmd := strings.Join(subCmdArgs, " ")
			if shCmd == "" {
				shCmd = "$SHELL"
			}
			envCmd := ""
			if len(envVars) > 0 {
				var quotedEnvVars []string
				for _, envVar := range envVars {
					quotedEnvVars = append(quotedEnvVars, fmt.Sprintf("'%s'", envVar))
				}
			envCmd = "env " + strings.Join(quotedEnvVars, " ") + " "
			}
			finalCmd := ""
			if shCmd == "$SHELL" {
				finalCmd = fmt.Sprintf("cd '%s'; exec %s%s", cwdRel, envCmd, shCmd)
			} else {
				escapedShCmd := strings.ReplaceAll(shCmd, "'", "'\\''")
				finalCmd = fmt.Sprintf("cd '%s'; exec %s sh -c '%s'", cwdRel, envCmd, escapedShCmd)
			}
			return "ssh", []string{host, "-t", finalCmd}, nil
		}

		// rsync

		if subCmd == "push" || subCmd == "pull" {
			if len(args) < 2 {
				return "", nil, fmt.Errorf("Usage: remote %s <file_path>", subCmd)
			}
			localFile := args[1]
			remoteFile := localFile
			if localFile[0] != '/' {
				remoteFile = filepath.Join(cwdRel, localFile)
			}
			localFileStat, err := os.Stat(localFile)
			localFileExists := false
			if err == nil {
				if localFileStat.IsDir() && localFile[len(localFile)-1] != '/' {
					localFile += "/"
					remoteFile += "/"
				}
				localFileExists = true
			}
			rsyncArgs := make([]string, 0, 8)
			for _, fname := range config.ExcludeFiles {
				rsyncArgs = append(rsyncArgs, "--exclude", fname)
			}

			if subCmd == "push" {
				if !localFileExists {
					return "", nil, fmt.Errorf("File not found: %q", localFile)
				}
				rsyncArgs = append(rsyncArgs, "-av", localFile, fmt.Sprintf("%s:%s", host, remoteFile))
				return "rsync", rsyncArgs, nil
			}

			if subCmd == "pull" {
				rsyncArgs = append(rsyncArgs, "-av", "--ignore-existing", fmt.Sprintf("%s:%s", host, remoteFile), localFile)
				return "rsync", rsyncArgs, nil
			}

		}

		// tunnel
		if subCmd == "tunnel" {
			if len(subCmdArgs) == 0 {
				return "", nil, errors.New("Usage: remote tunnel <port1> [port2]...")
			}
			sshArgs := []string{"-N"}
			if *isBackground {
				sshArgs = append(sshArgs, "-f")
			}
			for _, port := range subCmdArgs {
				sshArgs = append(sshArgs, "-L", fmt.Sprintf("%s:localhost:%s", port, port))
			}
			sshArgs = append(sshArgs, host)
			if *isVerbose {
				log.Printf("[DEBUG] Executing SSH tunnel command: %s %v", "ssh", sshArgs)
			}
			return "ssh", sshArgs, nil
		}

		return "", nil, fmt.Errorf("%q is not command", subCmd)
	}(host, flag.Args())
	if err != nil {
		return err
	}

	if cmdName == "" {
		return nil
	}

	// run command
	cmd := exec.Command(cmdName, cmdArg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if *isDryRun {
		fmt.Println(cmd.Args)
		return nil
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := _main(); err != nil {
		fmt.Println(err)
	}
}