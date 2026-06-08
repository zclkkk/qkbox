package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/getlantern/systray"
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

	// Unified exit — all shutdown paths converge here.
	var exitOnce sync.Once
	requestExit := func(wait bool) {
		exitOnce.Do(func() {
			if wait {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_ = inst.Shutdown(shutdownCtx)
				cancel()
			} else {
				inst.Close()
			}
			stop()
			if !*noTray {
				systray.Quit()
			}
		})
	}

	// When daemon exits (fatal error, signal, or user quit), clean up.
	go func() {
		if err := inst.Wait(); err != nil {
			log.Printf("daemon: %v", err)
		}
		requestExit(false)
	}()

	if *noTray {
		<-ctx.Done()
	} else {
		// Run tray on main goroutine (required by some platforms).
		trayRun(ctx, inst, requestExit)
	}
}
