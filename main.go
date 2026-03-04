package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/log"
	"github.com/daanv2/go-force-inline/cmd"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	root := cmd.NewRootCmd()

	if err := fang.Execute(
		context.Background(),
		root,
		fang.WithVersion(version),
		fang.WithCommit(commit),
	); err != nil {
		log.Fatal("error", "err", err)
		os.Exit(1)
	}
}
