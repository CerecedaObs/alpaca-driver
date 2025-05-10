package main

import (
	"alpaca/pkg/alpaca"
	"alpaca/pkg/alpaca/simulators"
	"alpaca/pkg/zro"
	"alpaca/templates"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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
		log.Fatalf("Error loading setup template: %v", err)
	}

	db, err := bolt.Open("alpaca.db", 0600, nil)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(c.String("broker"))
	opts.SetClientID("zro-alpaca")
	opts.SetUsername(c.String("username"))
	opts.SetPassword(c.String("password"))
	// opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
	// 	log.WithFields(log.Fields{
	// 		"topic":   msg.Topic(),
	// 		"payload": string(msg.Payload()),
	// 	}).Debug("Received message")
	// })

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	defer mqttClient.Disconnect(250)

	log.Info("Connected to MQTT broker")

	// TODO: Load ZRO configuration from database
	zroDome := zro.NewDome(mqttClient, zro.DefaultConfig, "/ZRO")

	dome := simulators.NewDomeSimulator(0, db, tmpl, log.WithField("device", "dome"))
	defer dome.Close()

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
		Addr:    fmt.Sprintf(":%d", c.Int("port")),
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
		zroDome.Run(ctx)
		wg.Done()
		log.Info("ZRO dome stopped")
	}()

	wg.Add(1)
	go func() {
		log.Debug("Server started")
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
			},
			&cli.StringFlag{
				Name:    "broker",
				Aliases: []string{"b"},
				Usage:   "MQTT broker address",
				Value:   "tcp://localhost:1883",
				EnvVars: []string{"MQTT_BROKER"},
			},
			&cli.StringFlag{
				Name:    "username",
				Aliases: []string{"u"},
				Usage:   "MQTT username",
				EnvVars: []string{"MQTT_USERNAME"},
			},
			&cli.StringFlag{
				Name:    "password",
				Aliases: []string{"pw"},
				Usage:   "MQTT password",
				EnvVars: []string{"MQTT_PASSWORD"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error: %v", err)
	}

}
