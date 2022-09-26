package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	remote_host_file := os.Getenv("REMOTE_WORKSPACE_FILE")
	if remote_host_file == "" {
		panic("Env not defined.")
	}
	fp, err := os.Open(remote_host_file)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	buf := make([]byte, 1024)
	_, err = fp.Read(buf)
	if err != nil {
		panic(err)
	}
	host := strings.Split(string(buf), "\n")[0]

	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println(path)
	cwd, err := filepath.Rel(os.Getenv("HOME"), path)
	fmt.Println(cwd)

	fmt.Printf("Connecting to %s\n", host)
	cmd := exec.Command("ssh", host, "-t", fmt.Sprintf("cd %s && bash", cwd))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		panic(err)
	}
}
