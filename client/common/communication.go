package common

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

type BetResponse struct {
	DNI string
	Num string
}

func (b Bet) SerializeBet() []byte {
	return []byte(fmt.Sprintf("%s;%s;%s;%s;%s;%s", b.AgenciaID, b.Nombre, b.Apellido, b.DNI, b.Nacimiento, b.Numero))
}

func serializeBetBatch(bets []Bet) (uint16, []byte, error) {
	var buf bytes.Buffer
	var betsBuf bytes.Buffer
	for i, b := range bets {
		betsBuf.Write(b.SerializeBet())
		if i < len(bets)-1 {
			betsBuf.WriteByte(',')
		}
	}

	totalLen := uint16(betsBuf.Len())
	const maxSize = 8192
	if totalLen > maxSize {
		return 0, nil, fmt.Errorf("serialized bets too large: %d bytes (max %d)", totalLen, maxSize)
	}

	buf.Write(betsBuf.Bytes())

	return totalLen, buf.Bytes(), nil
}

// Esta función agrega al payload el type header y el length prefix
func assembleMessage(msgType int, payload []byte) []byte {
	payloadLen := len(payload)

	buf := make([]byte, 1+2+payloadLen)
	buf[0] = byte(msgType)
	buf[1] = byte(payloadLen >> 8) // high
	buf[2] = byte(payloadLen)      // low
	copy(buf[3:], payload)

	return buf
}

func sendBytesToConnection(connection net.Conn, bytes []byte) int {
	totalSent := 0
	for totalSent < len(bytes) {
		n, err := connection.Write(bytes[totalSent:])
		if err != nil {
			return 0
		}
		totalSent += n
	}
	return totalSent
}

func uploadBetsBatch(connection net.Conn, datasetPath string, batchSize int) int {
	file, _ := os.Open(datasetPath)
	reader := csv.NewReader(file)
	uploaded := 0
	for {
		bets, finished := getBatchBetFromCSV(batchSize, reader)
		if finished && len(bets) == 0 {
			break
		}
		_, betsBytes, err := serializeBetBatch(bets)
		fullMessage := assembleMessage(1, betsBytes)
		sendBytesToConnection(connection, fullMessage)
		if err != nil {
			log.Errorf("action: send batch | result: fail | error: %v", err)
			return 0
		}
		// status, _ := receiveServerResponse(connection)
		// attempts := 0
		// for status != 5 && attempts < 3 { // reintentos por si falla el envio de una batch
		// 	sendBytesToConnection(connection, fullMessage)
		// 	status, _ = receiveServerResponse(connection)
		// 	attempts++
		// 	if attempts == 3 {
		// 		return -1
		// 	}
		// }
	}
	return uploaded
}

func sendFin(connection net.Conn) {
	finBytes := []byte("FIN")
	finMsg := assembleMessage(2, finBytes)
	sendBytesToConnection(connection, finMsg)
}

func sendWinnersRequest(connection net.Conn) {
	winnerBytes := []byte("WINNER")
	finMsg := assembleMessage(3, winnerBytes)
	sendBytesToConnection(connection, finMsg)
}

func receiveServerResponse(connection net.Conn) (int, error) {
	header := make([]byte, 3) // 1 byte tipo + 2 bytes length
	_, err := io.ReadFull(connection, header)

	attempts := 0
	for err != nil && attempts < 3 { //reintento leer hasta 3 veces sino tiro error
		header = make([]byte, 3) // 1 byte tipo + 2 bytes length
		_, err = io.ReadFull(connection, header)
		attempts++
		if attempts == 3 {
			return -1, fmt.Errorf("error leyendo header")
		}
	}

	msgType := header[0]
	if msgType == 4 {
		return -1, fmt.Errorf("los resultados todavía no están listos")
	}
	if msgType != 3 {
		return -1, fmt.Errorf("tipo de mensaje inesperado: %d", msgType)
	}

	msgLen := int(header[1])<<8 | int(header[2])
	if msgLen == 0 {
		return 0, nil
	}
	msg := make([]byte, msgLen)
	_, err = io.ReadFull(connection, msg)
	if err != nil {
		return 0, fmt.Errorf("error leyendo payload: %w", err)
	}
	payloadStr := string(msg)
	winnerStrs := strings.Split(payloadStr, ",")

	return len(winnerStrs), nil
}
