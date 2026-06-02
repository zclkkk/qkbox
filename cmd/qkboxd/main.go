package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zclkkk/qkbox/core/qkboxd"
	"github.com/zclkkk/qkbox/internal/ipc"
	"github.com/zclkkk/qkbox/shared/api"
)

func main() {
	version := flag.Bool("version", false, "print qkboxd version")
	endpoint := flag.Bool("endpoint", false, "print qkboxd IPC endpoint")
	flag.Parse()

	if *version {
		fmt.Println(api.QKBoxDVersion)
		return
	}
	if *endpoint {
		fmt.Println(ipc.Endpoint())
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := qkboxd.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
