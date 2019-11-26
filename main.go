package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		if st, ok := status.FromError(errors.Cause(err)); ok {
			switch st.Code() {
			case codes.NotFound:
				fmt.Println("Error: ", st.Message())
			case codes.Unavailable:
				fmt.Printf("Network error: %s\n", st.Message())
			default:
				fmt.Printf("%s: %s\n", st.Code(), st.Message())
			}

			if details := st.Details(); len(details) > 0 {
				fmt.Println("Details:")
				for _, msg := range st.Details() {
					fmt.Printf("- %s\n", msg)
				}
			}

			os.Exit(2)
		} else if _, ok := err.(cli.ExitCoder); !ok {
			fmt.Println(err)
			os.Exit(2)
		}
		cli.HandleExitCoder(cmd.Run(os.Args))
	}
}
