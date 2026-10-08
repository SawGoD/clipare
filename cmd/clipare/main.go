package main

import (
	"clipare"
	"clipare/internal/app"
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"clipare/internal/pairing"
	"clipare/internal/transport"
	"clipare/internal/ui"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

// AppKit must run on the initial process thread.
func init() { runtime.LockOSThread() }

func main() {
	if e := run(os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, "Clipare:", e)
		os.Exit(1)
	}
}
func run(args []string) error {
	status := len(args) > 0 && args[0] == "status"
	if status {
		args = args[1:]
	}
	fs := flag.NewFlagSet("clipare", flag.ContinueOnError)
	defaultPath, e := config.DefaultPath()
	if e != nil {
		return e
	}
	path := fs.String("config", defaultPath, "configuration file")
	headless := fs.Bool("headless", false, "run without tray or settings")
	debug := fs.Bool("debug", false, "debug metadata logs")
	ver := fs.Bool("version", false, "print version")
	if e := fs.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	if fs.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if *ver {
		fmt.Printf("Clipare %s (%s)\n", clipare.Version(), clipare.Commit)
		return nil
	}
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if !status && !*headless {
		return ui.Run(ctx, *path, log)
	}
	c, e := config.LoadMigrated(*path)
	if e != nil {
		return e
	}
	if status {
		client := transport.NewPeerClient(c)
		defer client.Close()
		return printStatus(ctx, client, c)
	}
	b, e := clipboard.New()
	if e != nil {
		return e
	}
	control := pairing.NewService(*path, c)
	control.DisablePairing()
	controlDone := make(chan struct{})
	go func() { defer close(controlDone); control.Propagate(ctx) }()
	defer func() { cancel(); <-controlDone }()
	s, e := app.StartWithControl(ctx, c, b, log, nil, control)
	if e != nil {
		return e
	}
	for {
		select {
		case <-ctx.Done():
			return s.Stop()
		case <-s.Done():
			return s.Err()
		case next := <-control.Updates:
			if e = s.Stop(); e != nil {
				return e
			}
			s, e = app.StartWithControl(ctx, next, b, log, nil, control)
			if e != nil {
				return e
			}
		}
	}
}
func printStatus(ctx context.Context, client *transport.Client, c config.Config) error {
	localURL := "http://" + c.ListenAddress()
	h, e := client.Health(ctx, localURL)
	if e != nil {
		return errors.New("local Clipare unavailable or authentication failed")
	}
	fmt.Printf("%s: %s (mode=%s)\n", h.Device, h.Status, h.Mode)
	for _, p := range c.Peers {
		h, e := client.HealthPeer(ctx, p.ID, p.URL())
		state := "offline"
		if e == nil && h.Device == p.ID {
			state = "online"
		}
		fmt.Printf("%s: %s\n", p.ID, state)
	}
	return nil
}
