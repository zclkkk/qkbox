package main

import (
	"context"
	"log"
	"time"

	"github.com/getlantern/systray"
	"github.com/zclkkk/qkbox/core/qkboxd"
	"github.com/zclkkk/qkbox/shared/api"
	"github.com/zclkkk/qkbox/shared/model"
)

// requestExitFunc is the unified shutdown callback. wait=true for user quit (blocks for cleanup).
type requestExitFunc func(wait bool)

// trayRun initialises the system tray and blocks until quit.
// Must be called from the main goroutine (required by some platforms).
func trayRun(ctx context.Context, inst *qkboxd.Instance, requestExit requestExitFunc) {
	systray.Run(
		func() { onReady(ctx, inst, requestExit) },
		func() { requestExit(false) },
	)
}

func onReady(ctx context.Context, inst *qkboxd.Instance, requestExit requestExitFunc) {
	// Status line (disabled, display-only).
	statusItem := systray.AddMenuItem("⏹ Stopped", "Engine status")
	statusItem.Disable()

	systray.AddSeparator()

	startItem := systray.AddMenuItem("▶ Start Engine", "Start the proxy engine")
	stopItem := systray.AddMenuItem("■ Stop Engine", "Stop the proxy engine")
	stopItem.Hide()

	systray.AddSeparator()

	windowItem := systray.AddMenuItem("⧉ Open Window", "Open the qkbox management window")

	systray.AddSeparator()

	quitItem := systray.AddMenuItem("Quit qkbox", "Shut down qkbox")

	// Engine state watcher — polls inst.EngineState() every 2 seconds.
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		lastState := ""
		for {
			select {
			case <-ticker.C:
				state := inst.EngineState()
				if state == lastState {
					continue
				}
				lastState = state
				switch model.EngineState(state) {
				case model.EngineStateStarted:
					statusItem.SetTitle("▶ Running")
					statusItem.SetTooltip("qkbox — Running")
					startItem.Hide()
					stopItem.Show()
				case model.EngineStateIdle, model.EngineStateUninitialized:
					statusItem.SetTitle("⏹ Stopped")
					statusItem.SetTooltip("qkbox — Stopped")
					startItem.Show()
					stopItem.Hide()
				case model.EngineStateFatal:
					statusItem.SetTitle("✖ Fatal")
					statusItem.SetTooltip("qkbox — Fatal error")
					startItem.Show()
					stopItem.Hide()
				default:
					statusItem.SetTitle("⏳ " + state)
					statusItem.SetTooltip("qkbox — " + state)
					startItem.Hide()
					stopItem.Hide()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Menu action handlers.
	go func() {
		for {
			select {
			case <-startItem.ClickedCh:
				_, _ = inst.Service.EngineStart(ctx, api.EngineStartRequest{})
			case <-stopItem.ClickedCh:
				_, _ = inst.Service.EngineStop(ctx, api.EngineStopRequest{})
			case <-windowItem.ClickedCh:
				openWindow(inst)
			case <-quitItem.ClickedCh:
				requestExit(true) // blocks until cleanup completes
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// openWindow sends a show event to the attached window, or spawns a new one.
func openWindow(inst *qkboxd.Instance) {
	sent, hasSession := inst.Service.NotifyWindowShow()
	if sent {
		return
	}
	if hasSession {
		log.Printf("open qkbox-window: session exists but window is unresponsive")
		return
	}
	if err := spawnQKBoxWindow(); err != nil {
		log.Printf("open qkbox-window: %v", err)
	}
}
