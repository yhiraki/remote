package command

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type SSHCommand struct{}

func (c *SSHCommand) Execute(ctx *Context) error {
	cmdName, cmdArgs, err := c.build(ctx.RemoteHost, ctx.Args, ctx.EnvVars, ctx.CwdRel)
	if err != nil {
		return err
	}
	return executeSubCommand(cmdName, cmdArgs, ctx.IsDryRun)
}

func (c *SSHCommand) build(remoteHost string, subCmdArgs, envVars []string, cwdRel string) (string, []string, error) {
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

func executeSubCommand(name string, args []string, isDryRun bool) error {
	cmd := exec.Command(name, args...)
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

