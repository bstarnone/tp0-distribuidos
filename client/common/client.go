package common

import (
	"net"
	"os"
	"strconv"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	BatchSize     string
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
	for msgID := 1; msgID <= 1; msgID++ {
		select {
		case <-sigChan:
			c.conn.Close()
			log.Infof("action: shutdown | result: success")
			return
		default:
			// Create the connection the server in every loop iteration. Send an
			c.createClientSocket()
			time.Sleep(5 * time.Second)
			// bet := getBetFromCSV("/dataset.csv")
			// uploadBet(c.conn, bet)
			// TODO: Modify the send to avoid short-read
			batchSize, _ := strconv.ParseInt(c.config.BatchSize, 10, 32)
			uploadBetsBatch(c.conn, "/dataset.csv", int(batchSize))
			sendFin(c.conn)
			time.Sleep(2 * time.Second)
			sendWinnersRequest(c.conn)
			// for i := 0; i < uploaded_bets_amount; i++ {
			response, _ := receiveWinners(c.conn)
			// if err != nil {
			// 	log.Errorf("action: receive_message | result: fail | client_id: %v | error: %v",
			// 		c.config.ID,
			// 		err,
			// 	)
			// 	return
			// }

			log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", response)
		}
		c.conn.Close()
		// }
	}
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
