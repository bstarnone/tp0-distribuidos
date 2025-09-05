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
	select {
	case <-sigChan:
		c.conn.Close()
		log.Infof("action: shutdown | result: success")
		return
	default:
		// Create the connection the server in every loop iteration. Send an
		c.createClientSocket()
		batchSize, _ := strconv.ParseInt(c.config.BatchSize, 10, 32)
		uploadBetsBatch(c.conn, "/dataset.csv", int(batchSize))
		sendFin(c.conn)
		sendWinnersRequest(c.conn)
		response, err := receiveWinners(c.conn)

		n := 1 * time.Second
		for response == -1 {
			log.Infof("action: consulta_ganadores | result: in progress | status: %v", err)
			time.Sleep(n)
			response, _ = receiveWinners(c.conn) //reintento
			n = n * 2                            //aumento el sleep por cada intento para no saturar el servidor (busy wait controlado)
		}

		log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", response)
	}
	c.conn.Close()
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
