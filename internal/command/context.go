package command

import "github.com/yhiraki/remote/internal/config"

type Context struct {
	Config       *config.Config
	RemoteHost   string
	Args         []string
	EnvVars      []string
	IsDryRun     bool
	IsBackground bool
	IsVerbose    bool
	CwdRel       string
}

