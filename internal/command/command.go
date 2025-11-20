package command

import (
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

	cmd, err := NewCommand(subCmd)
	if err != nil {
		return err
	}

	ctx := &Context{
		Config:       cfg,
		RemoteHost:   remoteHost,
		Args:         subCmdArgs,
		EnvVars:      envVars,
		IsDryRun:     isDryRun,
		IsBackground: isBackground,
		IsVerbose:    isVerbose,
		CwdRel:       cwdRel,
	}

	return cmd.Execute(ctx)
}
