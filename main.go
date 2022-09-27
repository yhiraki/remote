package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	// TODO: remote IP 情報をキャッシュしたい
	// cacheFile := filepath.Join(home, ".cache/remote/hostname")
	// if _, err := os.Stat(cacheFile); err != nil {
	// 	_, err := os.Create(cacheFile)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

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
	host := strings.TrimSuffix(string(out), "\n")

	// get relative current path
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	cwd, err := filepath.Rel(home, path)

	// build command args
	arg := make([]string, 0)
	if len(os.Args) == 1 {
		arg = append(arg, host, "-t", fmt.Sprintf("cd %s; exec %s", cwd, "bash"))
	} else if os.Args[1] == "sh" {
		arg = append(arg, host, "-t", fmt.Sprintf("cd %s; exec %s", cwd, strings.Join(os.Args[2:], " ")))
	} else {
		log.Fatal("Arg is not allowed")
	}

	// ssh connect
	fmt.Printf("Connecting to %s\n", host)
	cmd := exec.Command("ssh", arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}
