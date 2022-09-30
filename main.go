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
	var err error
	// ~/.config/remote/conf にIPを取得するためのコマンドを記載する
	configFile := filepath.Join(home, ".config", "remote", "conf")
	_, err = os.Stat(configFile)
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
	isCacheExpired := cacheFileState.ModTime().Add(timeBeforCacheExpies).Before(time.Now())
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

	// build command args
	arg := make([]string, 0)
	if len(os.Args) == 1 {
		arg = append(arg, host, "-t", fmt.Sprintf("cd %s; exec %s", cwd, "$SHELL"))
	} else if os.Args[1] == "sh" {
		arg = append(arg, host, "-t", fmt.Sprintf("cd %s; exec %s", cwd, strings.Join(os.Args[2:], " ")))
	} else {
		log.Fatal("Arg is not allowed")
	}

	// ssh connect
	fmt.Printf("Connecting to %s\n", host)
	cmd := exec.Command("ssh", arg...)
	fmt.Println(cmd.Args)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}
