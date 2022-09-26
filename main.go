package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var home string

func init() {
	var err error
	home, err = os.UserHomeDir()
	if err != nil {
		panic(err)
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

	buf := make([]byte, 1024)
	_, err = fp.Read(buf)
	if err != nil {
		panic(err)
	}
	shcmd := strings.Split(string(buf), "\n")[0]
	host := getHostname(shcmd)

	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	cwd, err := filepath.Rel(home, path)

	fmt.Printf("Connecting to %s\n", host)
	cmd := exec.Command("ssh", host, "-t", fmt.Sprintf("cd %s; exec $SHELL", cwd))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}

func getHostname(shcmd string) (host string) {
	// TODO: remote IP 情報をキャッシュしたい
	// cacheFile := filepath.Join(home, ".cache/remote/hostname")
	// if _, err := os.Stat(cacheFile); err != nil {
	// 	_, err := os.Create(cacheFile)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	cmd := exec.Command("sh", "-c", shcmd)
	out, err := cmd.Output()
	if err != nil {
		panic(err)
	}
	host = strings.TrimSuffix(string(out), "\n")
	return
}
