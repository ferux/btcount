package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/ferux/btcount/internal/btcount"
)

var (
	revision    string = "unknown"
	environment string = "development"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := btcount.ParseConfigFromEnv()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unable to parse config: %v\n", err)
		os.Exit(1)
	}

	err = app(ctx, cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unable to run the app: %v\n", err)
	}
}

func isDevelopment() bool {
	return strings.EqualFold(environment, "development")
}
