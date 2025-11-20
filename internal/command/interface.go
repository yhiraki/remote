package command

type Command interface {
	Execute(ctx *Context) error
}

