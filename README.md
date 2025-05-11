# ZRO-Alpaca

ZRO-Alpaca is a Go-based implementation of the ASCOM Alpaca protocol, designed to provide network-based control and automation for astronomical devices. This project enables seamless integration and management of observatory hardware such as domes and related equipment, supporting both real and simulated devices.

## Features

- Implements the ASCOM Alpaca REST API for device control
- Supports dome and store device types
- Includes simulators for testing and development
- Web-based setup and configuration interface

## Getting Started

1. Clone the repository.
2. Build the project using `make` or `go build`.
3. Run the server from the `cmd/zro-alpaca` directory.

## Running the Server

To start the ZRO-Alpaca server:

1. Build the project (if not already built):

   ```sh
   make
   # or
   go build -o tmp/main ./cmd/zro-alpaca
   ```

2. Run the server:

   ```sh
   ./tmp/main
   # or
   go run ./cmd/zro-alpaca -d
   ```

## Setup

You can setup the MQTT client by environment variables or by passing them as command line arguments. The following environment variables are used:

- `ALPACA_PORT` - The port on which the server will listen (default: `8090`)
- `MQTT_BROKER` - The MQTT broker address (default: `tcp://localhost:1883`)
- `MQTT_USERNAME` - The MQTT username (default: `""`)
- `MQTT_PASSWORD` - The MQTT password (default: `""`)

## Accessing the Setup Page

Once the server is running, open your web browser and navigate to:

[http://localhost:8090/setup](http://localhost:8090/setup)

This page provides a web-based interface for configuring the Alpaca server.

## Project Structure

- `cmd/zro-alpaca/` – Main application entry point
- `pkg/alpaca/` – Core Alpaca protocol implementation and device logic
- `pkg/alpaca/simulators/` – Simulated device implementations
- `templates/` – Web UI templates for device setup

## License

This project is licensed under the MIT License.
