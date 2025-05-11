package main

import (
	"alpaca/pkg/alpaca"
	"alpaca/pkg/drivers/dome_simulator"
	"alpaca/pkg/drivers/zro"
	"alpaca/templates"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	bolt "go.etcd.io/bbolt"
)

func run(c *cli.Context) error {
	if c.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	log.Info("ZRO Alpaca Server")

	tmpl, err := templates.LoadTemplates()
	if err != nil {
		return fmt.Errorf("failed to load templates: %v", err)
	}

	db, err := bolt.Open("alpaca.db", 0600, nil)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	store, err := alpaca.NewStore(db)
	if err != nil {
		return fmt.Errorf("failed to create store: %v", err)
	}

	simDome, err := dome_simulator.NewDomeSimulator(0, db, tmpl, log.WithField("device", "dome"))
	if err != nil {
		return fmt.Errorf("failed to create dome simulator: %v", err)
	}
	defer simDome.Close()

	zroDome, err := zro.NewDriver(1, db, tmpl, log.WithField("device", "zro"))
	if err != nil {
		return fmt.Errorf("failed to create ZRO dome: %v", err)
	}
	defer zroDome.Close()

	serverDesc := alpaca.ServerDescription{
		Name:                "ZRO Alpaca Server",
		Manufacturer:        "ZRO",
		ManufacturerVersion: "1.0",
		Location:            "ZRO",
	}

	devices := []alpaca.Device{
		simDome,
		zroDome,
	}
	server := alpaca.NewServer(serverDesc, devices, store, tmpl)

	mux := server.AddRoutes()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.Int("port")),
		Handler: mux,
	}

	// Channel to listen for interrupt or terminate signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		log.Debugf("Server started on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", srv.Addr, err)
		}
		wg.Done()
	}()

	// Create discovery responder
	discoveryLogger := log.WithField("component", "discovery")
	dr, err := alpaca.NewDiscoveryResponder("0.0.0.0", c.Int("port"), discoveryLogger)
	if err != nil {
		log.Fatalf("Failed to start discovery responder: %v", err)
	}

	wg.Add(1)
	go func() {
		if err := dr.Run(ctx); err != nil {
			log.Fatalf("Discovery responder failed: %v", err)
		}
		wg.Done()
		log.Debug("Discovery responder stopped")
	}()

	<-ctx.Done()

	log.Info("Shutting down server...")

	ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx2); err != nil {
		return fmt.Errorf("server forced to shutdown: %v", err)
	}

	wg.Wait()
	log.Info("Server stopped")
	return nil
}

func main() {
	app := cli.App{
		Name:  "ZRO Alpaca Server",
		Usage: "ZRO Alpaca Server",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug logging",
				Value:   false,
				EnvVars: []string{"DEBUG"},
			},
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "Port to listen on",
				Value:   8090,
				EnvVars: []string{"ALPACA_PORT"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error: %v", err)
	}

}
