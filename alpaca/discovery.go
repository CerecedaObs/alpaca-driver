package alpaca

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// DiscoveryResponder responds to Alpaca discovery requests.
type DiscoveryResponder struct {
	addr           string
	alpacaResponse string
	logger         log.FieldLogger
}

// NewDiscoveryResponder creates and starts a new discovery responder.
func NewDiscoveryResponder(addr string, port int, logger log.FieldLogger) (*DiscoveryResponder, error) {
	alpacaResponse := fmt.Sprintf(`{"AlpacaPort": %d}`, port)

	dr := DiscoveryResponder{
		addr:           addr,
		alpacaResponse: alpacaResponse,
		logger:         logger,
	}

	return &dr, nil
}

func (d *DiscoveryResponder) Run(ctx context.Context) error {
	buf := make([]byte, 1024)

	// Resolve the multicast address with port 32227
	deviceAddress, err := net.ResolveUDPAddr("udp", net.JoinHostPort(d.addr, "32227"))
	if err != nil {
		return fmt.Errorf("cannot resolve device address: %v", err)
	}

	// Create receive socket
	rSock, err := net.ListenUDP("udp", deviceAddress)
	if err != nil {
		return fmt.Errorf("cannot bind receive socket: %v", err)
	}
	defer rSock.Close()

	// Create a send socket bound to addr and an ephemeral port
	localAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(d.addr, "0"))
	if err != nil {
		return err
	}

	tSock, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return fmt.Errorf("cannot bind send socket: %v", err)
	}
	defer tSock.Close()

	d.logger.Debugf("Discovery responder started on %s", deviceAddress.String())
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Set a read deadline to periodically check for context cancellation
			rSock.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, addr, err := rSock.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout, continue
					continue
				}
				d.logger.Debugf("Error reading from socket: %v", err)
				continue
			}

			data := string(buf[:n])
			d.logger.Debugf("Received %s from %s", data, addr.String())

			if strings.Contains(data, "alpacadiscovery1") {
				if _, err := tSock.WriteToUDP([]byte(d.alpacaResponse), addr); err != nil {
					d.logger.Errorf("Error writing to socket: %v", err)
				}
			}
		}
	}
}

// func main() {

// 	// Replace "0.0.0.0" with appropriate IP address; port is the Alpaca port.
// 	_, err := NewDiscoveryResponder("0.0.0.0", 5555)
// 	if err != nil {
// 		log.Fatalf("Failed to start discovery responder: %v", err)
// 	}

// 	// Block forever, for example purposes.
// 	select {}
// }
