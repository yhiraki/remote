package command

import "fmt"

func NewCommand(subCmd string) (Command, error) {
	switch subCmd {
	case "ip":
		return &IPCommand{}, nil
	case "", "sh":
		return &SSHCommand{}, nil
	case "push", "pull":
		return &RsyncCommand{Direction: subCmd}, nil
	case "tunnel":
		return &TunnelCommand{}, nil
	default:
		return nil, fmt.Errorf("%q is not a valid command", subCmd)
	}
}

