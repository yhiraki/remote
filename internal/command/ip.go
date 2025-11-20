package command

import "fmt"

type IPCommand struct{}

func (c *IPCommand) Execute(ctx *Context) error {
	fmt.Println(ctx.RemoteHost)
	return nil
}

