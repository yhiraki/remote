package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/yhiraki/remote/internal/config"
	"github.com/yhiraki/remote/internal/host"
)

var (
	home string
	cwd  string
)

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

type stringSlice []string

func (i *stringSlice) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func _main() error {
	cfg, err := config.New()
	if err != nil {
		return err
	}

	configName := ".remoterc.json"
	if err := cfg.Load(configName); err != nil {
		log.Printf("%s could not parsed.", configName)
		return err
	}

	// create directories
	for _, d := range []string{cfg.CacheDir, cfg.ConfigDir} {
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
	h := cfg.Hostname
	if cfg.HostnameCommand != "" {
		h, err = host.Get(
			cfg.HostnameCommand,
			filepath.Join(cfg.CacheDir, "hostname"),
			cfg.CacheExpireMinutes,
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
	cmdName, cmdArg, err := func(remoteHost string, args []string) (string, []string, error) {
		subCmd := "sh"
		subCmdArgs := []string{}
		if len(args) > 0 {
			subCmd = args[0]
			subCmdArgs = args[1:]
		}

		// ip
		if subCmd == "ip" {
			fmt.Println(remoteHost)
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
			return "ssh", []string{remoteHost, "-t", finalCmd}, nil
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
			for _, fname := range cfg.ExcludeFiles {
				rsyncArgs = append(rsyncArgs, "--exclude", fname)
			}

			if subCmd == "push" {
				if !localFileExists {
					return "", nil, fmt.Errorf("File not found: %q", localFile)
				}
				rsyncArgs = append(rsyncArgs, "-av", localFile, fmt.Sprintf("%s:%s", remoteHost, remoteFile))
				return "rsync", rsyncArgs, nil
			}

			if subCmd == "pull" {
				rsyncArgs = append(rsyncArgs, "-av", "--ignore-existing", fmt.Sprintf("%s:%s", remoteHost, remoteFile), localFile)
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
			sshArgs = append(sshArgs, remoteHost)
			if *isVerbose {
				log.Printf("[DEBUG] Executing SSH tunnel command: %s %v", "ssh", sshArgs)
			}
			return "ssh", sshArgs, nil
		}

		return "", nil, fmt.Errorf("%q is not command", subCmd)
	}(h, flag.Args())
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
