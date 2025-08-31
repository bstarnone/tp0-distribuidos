package common

import (
	"net"
	"os"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.conn = conn
	return nil
}

// StartClientLoop Send messages to the client until some time threshold is met
func (c *Client) StartClientLoop(sigChan chan os.Signal) {
	// There is an autoincremental msgID to identify every message sent
	// Messages if the message amount threshold has not been surpassed
	for msgID := 1; msgID <= 100; msgID++ {
		select {
		case <-sigChan:
			c.conn.Close()
			log.Infof("action: shutdown | result: success")
			return
		default:
			bet := getBetFromCSV("/dataset.csv")
			// Create the connection the server in every loop iteration. Send an
			c.createClientSocket()

			uploadBet(c.conn, bet)

			response, err := receiveResponse(c.conn)
			// TODO: Modify the send to avoid short-read
			// _, err := bufio.NewReader(c.conn).ReadString('\n')

			c.conn.Close()

			if err != nil {
				log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
				return
			}

			log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
				response.DNI,
				response.Num,
			)
		}
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
