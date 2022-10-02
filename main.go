package main

import (
	"bufio"
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
	var cmd string
	arg := make([]string, 0)
	if len(os.Args) == 1 {
		cmd = "ssh"
		arg = append(arg, host, "-t", fmt.Sprintf("cd %s; exec %s", cwd, "$SHELL"))
	} else {
		switch os.Args[1] {
		case "sh":
			cmd = "ssh"
			arg = append(arg, host, "-t", fmt.Sprintf("cd %s; exec %s", cwd, strings.Join(os.Args[2:], " ")))
		case "push":
			cmd = "rsync"
			file := os.Args[2]
			remoteFile := file
			if file[0] != '/' {
				remoteFile = filepath.Join(cwd, file)
			}
			if fileStat, err := os.Stat(file); err != nil {
				log.Fatal(err)
			} else if fileStat.IsDir() && file[len(file)-1] != '/' {
				file += "/"
				remoteFile += "/"
			}
			arg = append(arg, "-av", file, fmt.Sprintf("%s:%s", host, remoteFile))
		case "pull":
			cmd = "rsync"
			file := os.Args[2]
			remoteFile := file
			if fileStat, err := os.Stat(file); err == nil {
				if fileStat.IsDir() && file[len(file)-1] != '/' {
					file += "/"
					remoteFile += "/"
				}
			}
			arg = append(arg, "-av", "--ignore-existing", fmt.Sprintf("%s:%s", host, remoteFile), file)
		default:
			log.Fatal("Arg is not allowed")
		}
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
		cmd := exec.Command(cmd, arg...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Println(cmd.Args)
		fmt.Printf("Connecting to %s\n", host)
		if err = cmd.Run(); err != nil {
			continue
		}
		break
	}
}
