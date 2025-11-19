package command

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yhiraki/remote/internal/config"
)

// Run executes the appropriate subcommand based on the provided arguments.
func Run(cfg *config.Config, remoteHost string, args []string, envVars []string, isDryRun, isBackground, isVerbose bool, cwdRel string) error {
	subCmd := "sh"
	subCmdArgs := []string{}
	if len(args) > 0 {
		subCmd = args[0]
		subCmdArgs = args[1:]
	}

	var cmdName string
	var cmdArgs []string
	var err error

	switch subCmd {
	case "ip":
		fmt.Println(remoteHost)
		return nil
	case "", "sh":
		cmdName, cmdArgs, err = buildSSHCommand(remoteHost, subCmdArgs, envVars, cwdRel)
	case "push", "pull":
		cmdName, cmdArgs, err = buildRsyncCommand(subCmd, remoteHost, subCmdArgs, cfg.ExcludeFiles, cwdRel)
	case "tunnel":
		cmdName, cmdArgs, err = buildTunnelCommand(remoteHost, subCmdArgs, isBackground, isVerbose)
	default:
		return fmt.Errorf("%q is not a valid command", subCmd)
	}

	if err != nil {
		return err
	}

	if cmdName == "" {
		return nil
	}

	// run command
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if isDryRun {
		fmt.Println(cmd.Args)
		return nil
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func buildSSHCommand(remoteHost string, subCmdArgs, envVars []string, cwdRel string) (string, []string, error) {
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

func buildRsyncCommand(subCmd, remoteHost string, subCmdArgs, excludeFiles []string, cwdRel string) (string, []string, error) {
	if len(subCmdArgs) < 1 {
		return "", nil, fmt.Errorf("Usage: remote %s <file_path>", subCmd)
	}
	localFile := subCmdArgs[0]
	remoteFile := localFile
	if !filepath.IsAbs(localFile) {
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
	for _, fname := range excludeFiles {
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
	return "", nil, errors.New("unsupported rsync subcommand")
}

func buildTunnelCommand(remoteHost string, subCmdArgs []string, isBackground, isVerbose bool) (string, []string, error) {
	if len(subCmdArgs) == 0 {
		return "", nil, errors.New("Usage: remote tunnel <port1> [port2]...")
	}
	sshArgs := []string{"-N"}
	if isBackground {
		sshArgs = append(sshArgs, "-f")
	}
	for _, port := range subCmdArgs {
		sshArgs = append(sshArgs, "-L", fmt.Sprintf("%s:localhost:%s", port, port))
	}
	sshArgs = append(sshArgs, remoteHost)
	if isVerbose {
		log.Printf("[DEBUG] Executing SSH tunnel command: %s %v", "ssh", sshArgs)
	}
	return "ssh", sshArgs, nil
}
