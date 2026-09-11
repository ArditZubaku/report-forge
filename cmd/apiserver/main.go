package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ArditZubaku/async-api/apiserver"
	"github.com/ArditZubaku/async-api/config"
	"github.com/ArditZubaku/async-api/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	conf, err := config.New()
	if err != nil {
		return err
	}

	db, err := store.NewPG(conf.DatabaseURL())
	if err != nil {
		return err
	}

	dataStore := store.New(db)

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)

	server := apiserver.New(conf, logger, dataStore)
	return server.Start(ctx)
}
