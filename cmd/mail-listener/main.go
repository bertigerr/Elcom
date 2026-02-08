package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"elcom/internal/config"
	"elcom/internal/listener"
	"elcom/internal/storage"
)

func main() {
	cfg, err := config.Load()
	must(err)

	db, err := storage.Open(cfg.DBPath)
	must(err)
	defer db.Close()

	svc := listener.NewService(db, cfg)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	must(svc.Run(ctx))
}

func must(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
