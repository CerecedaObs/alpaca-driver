package main

import (
	"alpaca/alpaca"
	"alpaca/alpaca/simulators"
	"alpaca/templates"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

func main() {
	port := flag.Int("port", 8090, "Port to listen on")
	flag.Parse()

	log.SetLevel(log.DebugLevel)
	log.Info("ZRO Alpaca Server")

	tmpl, err := templates.LoadTemplates()
	if err != nil {
		log.Fatalf("Error loading setup template: %v", err)
	}

	db, err := bolt.Open("alpaca.db", 0600, nil)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	dome := simulators.NewDomeSimulator(0, db, tmpl, log.WithField("device", "dome"))

	serverDesc := alpaca.ServerDescription{
		Name:                "ZRO Alpaca Server",
		Manufacturer:        "ZRO",
		ManufacturerVersion: "1.0",
		Location:            "ZRO",
	}

	store, err := alpaca.NewStore(db)
	if err != nil {
		log.Fatalf("Error creating store: %v", err)
	}

	server := alpaca.NewServer(serverDesc, []alpaca.Device{dome}, store, tmpl)

	mux := server.AddRoutes()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	// Channel to listen for interrupt or terminate signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// stop := make(chan os.Signal, 1)
	// signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		log.Debug("Server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", srv.Addr, err)
		}
		wg.Done()
		log.Debug("Server stopped")
	}()

	// Create discovery responder
	discoveryLogger := log.WithField("component", "discovery")
	dr, err := alpaca.NewDiscoveryResponder("0.0.0.0", *port, discoveryLogger)
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
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	wg.Wait()
	log.Info("Server stopped")
}
