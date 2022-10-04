package main

import (
	"bufio"
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
	cacheDir string
)

func init() {
	var err error

	home, err = os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	cacheDir = filepath.Join(home, ".cache/remote")
	_, err = os.Stat(cacheDir)
	if err != nil {
		if err = os.MkdirAll(cacheDir, 0o705); err != nil {
			panic(err)
		}
	}
}

func main() {
	// ~/.config/remote/conf にIPを取得するためのコマンドを記載する
	configFile := filepath.Join(home, ".config", "remote", "conf")
	_, err := os.Stat(configFile)
	if err != nil {
		panic(err)
	}
	fp, err := os.Open(configFile)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	// command line parsing
	isDryRun := flag.Bool("dry-run", false, "dry run")
	flag.Parse()

	// get remote hostname and cache
	timeBeforCacheExpies := 24 * time.Hour
	cacheFile := filepath.Join(cacheDir, "hostname")
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
		buf := make([]byte, 1024)
		_, err = fp.Read(buf)
		if err != nil {
			panic(err)
		}
		shcmd := strings.Split(string(buf), "\n")[0]
		out, err := exec.Command("sh", "-c", shcmd).Output()
		if err != nil {
			panic(err)
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
	host := string(sc.Text())

	// get relative current path
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	cwd, err := filepath.Rel(home, path)
	if err != nil {
		panic(err)
	}

	// build command args
	cmd, cmdArg, err := func(args []string) (string, []string, error) {
		// ssh

		if len(args) == 0 {
			return "ssh", []string{host, "-t", fmt.Sprintf("cd %s; exec %s", cwd, "$SHELL")}, nil
		}

		subCmd := args[0]

		if subCmd == "sh" {
			return "ssh", []string{host, "-t", fmt.Sprintf("cd %s; exec %s", cwd, strings.Join(args[1:], " "))}, nil
		}

		// rsync

		localFile := args[1]
		remoteFile := localFile
		if localFile[0] != '/' {
			remoteFile = filepath.Join(cwd, localFile)
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
		rsyncOptions := []string{"--exclude", "node_modules", "--exclude", ".venv"}

		if subCmd == "push" {
			if !localFileExists {
				return "", nil, errors.New(fmt.Sprintf("File not found: %q", localFile))
			}
			rsyncArgs = append(rsyncArgs, "-av", localFile, fmt.Sprintf("%s:%s", host, remoteFile))
			rsyncArgs = append(rsyncArgs, rsyncOptions...)
			return "rsync", rsyncArgs, nil
		}

		if subCmd == "pull" {
			rsyncArgs = append(rsyncArgs, "-av", "--ignore-existing", fmt.Sprintf("%s:%s", host, remoteFile), localFile)
			rsyncArgs = append(rsyncArgs, rsyncOptions...)
			return "rsync", rsyncArgs, nil
		}

		return "", nil, errors.New(fmt.Sprintf("%q is not command", subCmd))
	}(flag.Args())
	if err != nil {
		log.Fatal(err)
	}

	// ssh connect
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
		fmt.Println(cmd.Args)
		if *isDryRun {
			break
		}
		fmt.Printf("Connecting to %s\n", host)
		if err := cmd.Run(); err != nil {
			continue
		}
		break
	}
}
