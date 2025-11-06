package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	wd := strings.Split(cwd, "/")
	for ; len(wd) > 0; wd = wd[:len(wd)-1] {
		path := filepath.Join(wd...)
		path = filepath.Join("/", path, name)
		if s, err := os.Stat(path); err != nil {
			continue
		} else if !s.IsDir() {
			return path, nil
		}
		break
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
	cmd string, cacheFile string, timeBeforCacheExpies time.Duration,
) (string, error) {
	cacheFileState, err := os.Stat(cacheFile)
	isCacheExpired := true
	if err == nil {
		isCacheExpired = cacheFileState.ModTime().Add(timeBeforCacheExpies).Before(time.Now())
	}
	if err != nil || isCacheExpired {
		f, err := os.Create(cacheFile)
		if err != nil {
			return "", errors.New("Could not create hostname cachefile")
		}
		defer f.Close()

		// get remote hostname
		shcmd := strings.Split(cmd, "\n")[0]
		parts := strings.Fields(shcmd) // コマンドと引数をスペースで分割
		if len(parts) == 0 {
			return "", errors.New("Hostname command is empty")
		}
		cmdName := parts[0]
		cmdArgs := parts[1:]
		out, err := exec.Command(cmdName, cmdArgs...).Output()
		if err != nil {
			return "", errors.New("Could not get hostname")
		}
		f.WriteString(strings.TrimSuffix(string(out), "\n"))
	}

	// get cached remote hostname
	f, err := os.Open(cacheFile)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Could not open hostname cachefile: %s", cacheFile))
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return "", errors.New("Host not cached")
	}

	host := string(sc.Text())
	return host, nil
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
	isDryRun := flag.Bool("dry-run", false, "dry run")
	flag.Parse()

	// get hostname
	host := config.Hostname
	if config.HostnameCommand != "" {
		timeBeforCacheExpies := time.Duration(config.CacheExpireMinutes) * time.Minute
		host, err = getRemoteHostname(
			config.HostnameCommand,
			filepath.Join(config.CacheDir, "hostname"),
			timeBeforCacheExpies)
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
	cmdName, cmdArg, err := func(args []string) (string, []string, error) {
		subCmd := "sh"
		subCmdArgs := []string{}
		if len(args) > 0 {
			subCmd = args[0]
			subCmdArgs = args[1:]
		}

		// ssh

		if subCmd == "" || subCmd == "sh" {
			shCmd := strings.Join(subCmdArgs, " ")
			if shCmd == "" {
				shCmd = "$SHELL"
			}
			return "ssh", []string{host, "-t", fmt.Sprintf("cd '%s'; exec %s", cwdRel, shCmd)}, nil
		}

		// rsync

		if subCmd == "push" || subCmd == "pull" {
			if len(args) < 2 {
				return "", nil, errors.New(fmt.Sprintf("Usage: remote %s <file_path>", subCmd))
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
					return "", nil, errors.New(fmt.Sprintf("File not found: %q", localFile))
				}
				rsyncArgs = append(rsyncArgs, "-av", localFile, fmt.Sprintf("%s:%s", host, remoteFile))
				return "rsync", rsyncArgs, nil
			}

			if subCmd == "pull" {
				rsyncArgs = append(rsyncArgs, "-av", "--ignore-existing", fmt.Sprintf("%s:%s", host, remoteFile), localFile)
				return "rsync", rsyncArgs, nil
			}

		}

		return "", nil, errors.New(fmt.Sprintf("%q is not command", subCmd))
	}(flag.Args())
	if err != nil {
		return err
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
	cmd.Run()
	return nil
}

func main() {
	if err := _main(); err != nil {
		fmt.Println(err)
	}
}
