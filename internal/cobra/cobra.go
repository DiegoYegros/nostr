package cobra

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type PositionalArgs func(cmd *Command, args []string) error

type Command struct {
	Use   string
	Short string
	Long  string

	Args PositionalArgs
	Run  func(cmd *Command, args []string)
	RunE func(cmd *Command, args []string) error

	parent   *Command
	commands []*Command
	flags    *flag.FlagSet
}

func (c *Command) AddCommand(commands ...*Command) {
	for _, cmd := range commands {
		cmd.parent = c
		c.commands = append(c.commands, cmd)
	}
}

func (c *Command) Execute() error {
	args := os.Args[1:]
	return c.execute(args)
}

func (c *Command) execute(args []string) error {
	if len(args) > 0 {
		if child := c.findSubcommand(args[0]); child != nil {
			return child.execute(args[1:])
		}
	}

	if c.flags != nil {
		c.flags.SetOutput(io.Discard)
		if err := c.flags.Parse(args); err != nil {
			return err
		}
		args = c.flags.Args()
	}

	if c.Args != nil {
		if err := c.Args(c, args); err != nil {
			return err
		}
	}

	if c.RunE != nil {
		return c.RunE(c, args)
	}

	if c.Run != nil {
		c.Run(c, args)
	}

	return nil
}

func (c *Command) findSubcommand(name string) *Command {
	for _, cmd := range c.commands {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func (c *Command) Name() string {
	parts := strings.Split(c.Use, " ")
	if len(parts) > 0 {
		return parts[0]
	}
	return c.Use
}

func (c *Command) Flags() *flag.FlagSet {
	if c.flags == nil {
		c.flags = flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	}
	return c.flags
}

func (c *Command) Help() error {
	fmt.Fprintf(os.Stdout, "Usage: %s\n", c.Use)
	if c.Short != "" {
		fmt.Fprintf(os.Stdout, "\n%s\n", c.Short)
	}
	if len(c.commands) > 0 {
		fmt.Fprintln(os.Stdout, "\nAvailable Commands:")
		for _, cmd := range c.commands {
			fmt.Fprintf(os.Stdout, "  %s\t%s\n", cmd.Name(), cmd.Short)
		}
	}
	return nil
}

func ArbitraryArgs(cmd *Command, args []string) error { return nil }

func ExactArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("%s requires %d arg(s)", cmd.Name(), n)
		}
		return nil
	}
}
