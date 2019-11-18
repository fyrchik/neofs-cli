package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

func main() {
	cmd := cli.NewApp()
	cmd.Name = Name
	cmd.Usage = "Example of tool that provides basic interactions with NeoFS"
	cmd.Version = fmt.Sprintf("%s (%s)", Version, Build)
	cmd.Commands = commands()
	cmd.Flags = getFlags(Global)
	cmd.Before = beforeAction

	if err := cmd.Run(os.Args); err != nil {
		if _, ok := err.(cli.ExitCoder); !ok {
			fmt.Println(err)
			os.Exit(2)
		}
		cli.HandleExitCoder(cmd.Run(os.Args))
	}
}
