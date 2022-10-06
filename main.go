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
	home     string
	cwd      string
	cacheDir string
)

type Config struct {
	Hostname        string   `json:"hostname"`
	HostnameCommand string   `json:"hostnameCommand"`
	ExcludeFiles    []string `json:"excludeFiles"`
}

// Find nearest config file path
func findConfigFile() (filePath string) {
	configName := ".remoterc.json"
	filePath = filepath.Join(home, ".config", "remote", configName)
	wd := strings.Split(cwd, "/")
	for ; len(wd) > 0; wd = wd[:len(wd)-1] {
		path := filepath.Join(wd...)
		path = filepath.Join("/", path, configName)
		if s, err := os.Stat(path); err != nil {
			continue
		} else if !s.IsDir() {
			filePath = path
		}
		break
	}
	return
}

func parseConfigJson(config *Config) {
	fp, err := os.Open(findConfigFile())
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	if err := json.NewDecoder(fp).Decode(&config); err != nil {
		log.Fatal(err)
	}
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

	cacheDir = filepath.Join(home, ".cache/remote")
	_, err = os.Stat(cacheDir)
	if err != nil {
		if err = os.MkdirAll(cacheDir, 0o705); err != nil {
			log.Fatal(err)
		}
	}
}

func getRemoteHostname(cmd string) (host string) {
	// get remote hostname and cache
	timeBeforCacheExpies := 24 * time.Hour
	cacheFile := filepath.Join(cacheDir, "hostname") // TODO: config file の場所によって変える
	cacheFileState, err := os.Stat(cacheFile)
	isCacheExpired := true
	if err == nil {
		isCacheExpired = cacheFileState.ModTime().Add(timeBeforCacheExpies).Before(time.Now())
	}
	if err != nil || isCacheExpired {
		f, err := os.Create(cacheFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		// get remote hostname
		shcmd := strings.Split(cmd, "\n")[0]
		out, err := exec.Command("sh", "-c", shcmd).Output()
		if err != nil {
			log.Fatal(err)
		}
		f.WriteString(strings.TrimSuffix(string(out), "\n"))
	}

	// get cached remote hostname
	f, err := os.Open(cacheFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		log.Fatal("Host not cached")
	}

	host = string(sc.Text())
	return
}

func main() {
	var config Config
	parseConfigJson(&config)

	// command line parsing
	isDryRun := flag.Bool("dry-run", false, "dry run")
	flag.Parse()

	// get hostname
	host := config.Hostname
	if config.HostnameCommand != "" {
		host = getRemoteHostname(config.HostnameCommand)
	}

	// get relative current path
	cwdRel, err := filepath.Rel(home, cwd)
	if err != nil {
		log.Fatal(err)
	}

	// build command args
	cmd, cmdArg, err := func(args []string) (string, []string, error) {
		// ssh

		if len(args) == 0 {
			return "ssh", []string{host, "-t", fmt.Sprintf("cd %s; exec %s", cwdRel, "$SHELL")}, nil
		}

		subCmd := args[0]

		if subCmd == "sh" {
			return "ssh", []string{host, "-t", fmt.Sprintf("cd %s; exec %s", cwdRel, strings.Join(args[1:], " "))}, nil
		}

		// rsync

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

		return "", nil, errors.New(fmt.Sprintf("%q is not command", subCmd))
	}(flag.Args())
	if err != nil {
		log.Fatal(err)
	}

	// run command
	waitSeconds := 1
	maxRetry := 3
	for i := 0; i <= maxRetry; i++ {
		if i > 0 {
			log.Printf("%s\nRetry after %d seconds.", err, waitSeconds)
			time.Sleep(time.Second * time.Duration(waitSeconds))
			waitSeconds *= 2
		}
		cmd := exec.Command(cmd, cmdArg...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if *isDryRun {
			fmt.Println(cmd.Args)
			break
		}
		// exit code 255 is ssh connection error
		if err := cmd.Run(); err != nil && cmd.ProcessState.ExitCode() == 255 {
			continue
		}
		break
	}
}
