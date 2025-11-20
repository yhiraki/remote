package command

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type RsyncCommand struct {
	Direction string // "push" or "pull"
}

func (c *RsyncCommand) Execute(ctx *Context) error {
	cmdName, cmdArgs, err := c.build(ctx.RemoteHost, ctx.Args, ctx.Config.ExcludeFiles, ctx.CwdRel)
	if err != nil {
		return err
	}
	return executeSubCommand(cmdName, cmdArgs, ctx.IsDryRun)
}

func (c *RsyncCommand) build(remoteHost string, subCmdArgs, excludeFiles []string, cwdRel string) (string, []string, error) {
	if len(subCmdArgs) < 1 {
		return "", nil, fmt.Errorf("Usage: remote %s <file_path>", c.Direction)
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

	if c.Direction == "push" {
		if !localFileExists {
			return "", nil, fmt.Errorf("File not found: %q", localFile)
		}
		rsyncArgs = append(rsyncArgs, "-av", localFile, fmt.Sprintf("%s:%s", remoteHost, remoteFile))
		return "rsync", rsyncArgs, nil
	}

	if c.Direction == "pull" {
		rsyncArgs = append(rsyncArgs, "-av", "--ignore-existing", fmt.Sprintf("%s:%s", remoteHost, remoteFile), localFile)
		return "rsync", rsyncArgs, nil
	}
	return "", nil, errors.New("unsupported rsync subcommand")
}

