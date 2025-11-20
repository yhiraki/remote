package command

import (
	"errors"
	"fmt"
	"log"
)

type TunnelCommand struct{}

func (c *TunnelCommand) Execute(ctx *Context) error {
	cmdName, cmdArgs, err := c.build(ctx.RemoteHost, ctx.Args, ctx.IsBackground, ctx.IsVerbose)
	if err != nil {
		return err
	}
	return executeSubCommand(cmdName, cmdArgs, ctx.IsDryRun)
}

func (c *TunnelCommand) build(remoteHost string, subCmdArgs []string, isBackground, isVerbose bool) (string, []string, error) {
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

