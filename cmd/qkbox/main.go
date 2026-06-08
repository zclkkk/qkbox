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
	version := flag.Bool("version", false, "print qkbox version")
	endpoint := flag.Bool("endpoint", false, "print qkbox IPC endpoint")
	noTray := flag.Bool("no-tray", false, "run without system tray (headless mode)")
	flag.Parse()

	if *version {
		fmt.Println(api.AppVersion)
		return
	}
	if *endpoint {
		fmt.Println(ipc.Endpoint())
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	inst, err := qkboxd.Start(ctx, qkboxd.StartOpts{})
	if err != nil {
		log.Fatal(err)
	}

	// Wait for daemon exit in background; ensure tray quits when daemon exits.
	go func() {
		if err := inst.Wait(); err != nil {
			log.Printf("daemon: %v", err)
		}
	}()

	if *noTray {
		// Headless mode: just wait for signal.
		<-ctx.Done()
	} else {
		// Run tray on main goroutine (required by some platforms).
		trayRun(ctx, inst)
	}
}
