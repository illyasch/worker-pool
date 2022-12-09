package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf/v3"

	"github.com/illyasch/worker-pool/examples/password-bcrypt-service/handlers"
	"github.com/illyasch/worker-pool/pool"
)

const (
	configPrefix = "BCRYPT"
)

type config struct {
	conf.Version
	APIHost         string        `conf:"default:0.0.0.0:3000"`
	NumWorkers      int           `conf:"default:10"`
	ShutdownTimeout time.Duration `conf:"default:20s"`
	BusyTimeout     time.Duration `conf:"default:100ms"`
}

func main() {
	l := log.New(os.Stdout, configPrefix+": ", log.LstdFlags)

	if err := run(l); err != nil {
		l.Fatal("startup", "ERROR", err)
	}
}

func run(logger *log.Logger) error {
	cfg, err := parseConfig(configPrefix, logger)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	// =========================================================================
	// App Starting

	logger.Println("starting service")
	defer logger.Println("shutdown complete")

	// Start worker pool.
	workers := pool.NewNonBlocking[string](cfg.NumWorkers)
	workers.Run(context.Background())
	defer workers.Stop()

	// =========================================================================
	// Start API Service

	logger.Println("startup", "status", "initializing API support")

	// Make a channel to listen for an interrupt or terminate signal from the OS.
	// Use a buffered channel because the signal package requires it.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Construct the mux for the API calls.
	apiMux := handlers.APIConfig{
		BusyTimeout: cfg.BusyTimeout,
		Log:         logger,
		Workers:     workers,
	}.Router()

	// Construct a server to service the requests against the mux.
	srv := http.Server{
		Addr:     cfg.APIHost,
		Handler:  apiMux,
		ErrorLog: logger,
	}

	// Make a channel to listen for errors coming from the listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	// Start the service listening for srv requests.
	go func() {
		logger.Println("startup", "status", "srv router started", "host", srv.Addr)
		serverErrors <- srv.ListenAndServe()
	}()

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		logger.Println("shutdown", "status", "shutdown started", "signal", sig)
		defer logger.Println("shutdown", "status", "shutdown complete", "signal", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		// Asking listener to shut down and shed load.
		if err := srv.Shutdown(ctx); err != nil {
			if cErr := srv.Close(); cErr != nil {
				logger.Println("shutdown", "ERROR", fmt.Errorf("server close: %w", cErr))
			}
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

func parseConfig(prefix string, logger *log.Logger) (config, error) {
	cfg := config{
		Version: conf.Version{
			Desc: "Copyright Ilya Scheblanov",
		},
	}

	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		fmt.Println(help)
		return cfg, err
	}

	out, err := conf.String(&cfg)
	if err != nil {
		return cfg, fmt.Errorf("generating config for output: %w", err)
	}
	logger.Println("startup", "config", out)

	return cfg, nil
}
