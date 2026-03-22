package main

import (
	"context"
	"fmt"
	"os"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/graphcmd"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := graphcmd.Execute(context.Background(), os.Args[1:], cwd, os.Stdin, os.Stdout, os.Stderr); err != nil {
		if !cmdsupport.IsSilentError(err) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
