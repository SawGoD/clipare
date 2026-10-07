package main

import (
	"clipare/internal/clipboard"
	"clipare/internal/config"
	"clipare/internal/security"
	clipsync "clipare/internal/sync"
	"clipare/internal/transport"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "0.1.0-dev"

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
	path := fs.String("config", "config.yaml", "configuration file")
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
		fmt.Println("Clipare", version)
		return nil
	}
	c, e := config.Load(*path)
	if e != nil {
		return e
	}
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	secret := security.StaticSecret(c.Security.Secret)
	client := transport.NewClient(secret)
	defer client.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if status {
		return printStatus(ctx, client, c)
	}
	b, e := clipboard.New()
	if e != nil {
		return e
	}
	listener, e := net.Listen("tcp", c.ListenAddress())
	if e != nil {
		return errors.New("cannot listen on configured address")
	}
	defer listener.Close()
	workers := transport.StartWorkers(ctx, client, c.Peers, log)
	defer workers.Wait()
	defer cancel()
	manager := clipsync.NewManager(b, c.Device.ID, c.Mode(), log, workers.Broadcast)
	if e = manager.Initialize(); e != nil {
		return errors.New("cannot initialize clipboard")
	}
	server := &http.Server{Handler: transport.Handler(c, secret, manager, log), ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192, BaseContext: func(net.Listener) context.Context { return ctx }}
	results := make(chan error, 2)
	events := make(chan struct{}, 1)
	go func() {
		e := server.Serve(listener)
		if errors.Is(e, http.ErrServerClosed) {
			e = nil
		}
		results <- e
	}()
	go func() { results <- b.Watch(ctx, events) }()
	log.Info("Clipare started", "device", c.Device.ID, "listen", c.ListenAddress(), "mode", c.Mode())
	var result error
	completed := 0
running:
	for {
		select {
		case <-ctx.Done():
			break running
		case e := <-results:
			result = e
			completed++
			break running
		case <-events:
			manager.LocalChanged()
		}
	}
	cancel()
	shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if e = server.Shutdown(shutdown); e != nil {
		server.Close()
		if result == nil {
			result = errors.New("HTTP shutdown timed out")
		}
	}
	for completed < 2 {
		e := <-results
		completed++
		if result == nil && e != nil {
			result = e
		}
	}
	log.Info("Clipare stopped")
	return result
}
func printStatus(ctx context.Context, client *transport.Client, c config.Config) error {
	localURL := "http://" + c.ListenAddress()
	h, e := client.Health(ctx, localURL)
	if e != nil {
		return errors.New("local Clipare unavailable or authentication failed")
	}
	fmt.Printf("%s: %s (mode=%s)\n", h.Device, h.Status, h.Mode)
	for _, p := range c.Peers {
		h, e := client.Health(ctx, p.URL())
		state := "offline"
		if e == nil && h.Device == p.ID {
			state = "online"
		}
		fmt.Printf("%s: %s\n", p.ID, state)
	}
	return nil
}
