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
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

func main() {
	port := flag.Uint("port", 8080, "Port to listen on")
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
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Debug("Server started")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", srv.Addr, err)
		}
	}()

	<-stop // Wait for interrupt signal

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped")
}
