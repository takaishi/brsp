package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/takaishi/brsp"
)

var Version = "dev"
var Revision = "HEAD"

func init() {
	Version = Version
	Revision = Revision
}

func main() {
	ctx := context.TODO()
	ctx, stop := signal.NotifyContext(ctx, []os.Signal{os.Interrupt}...)
	defer stop()
	if err := brsp.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
	go func() {
		<-ctx.Done()
		_, cancel := context.WithCancel(context.Background())
		defer cancel()
	}()
}
