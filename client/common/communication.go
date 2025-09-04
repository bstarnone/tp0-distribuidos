package common

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// type BetMessage struct {
// 	data []byte
// 	len  uint32
// }

type BetResponse struct {
	DNI string
	Num string
}

func (b Bet) SerializeBet() []byte {
	return []byte(fmt.Sprintf("%s;%s;%s;%s;%s;%s", b.AgenciaID, b.Nombre, b.Apellido, b.DNI, b.Nacimiento, b.Numero))
}

// func uploadBet(connection net.Conn, bet Bet) {
// 	data := bet.SerializeBet()
// 	msg := BetMessage{
// 		data: data,
// 		len:  uint32(len(data)),
// 	}
// 	// TODO: cambiar la lógica para recibir un BetMessage en vez de un Bet

// 	msg_buf := new(bytes.Buffer)
// 	binary.Write(msg_buf, binary.BigEndian, msg.len)
// 	msg_buf.Write(msg.data)

// 	full_msg := msg_buf.Bytes()
// 	total_sent := 0

// 	for total_sent < len(full_msg) {
// 		n, err := connection.Write(full_msg[total_sent:])
// 		if err != nil {
// 			// fmt.Println("[ERROR] Error enviando bet al servidor:", err)
// 			return
// 		}
// 		total_sent += n
// 	}
// }

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

	// if err := binary.Write(&buf, binary.BigEndian, totalLen); err != nil {
	// 	return 0, nil, fmt.Errorf("error escribiendo length prefix: %w", err)
	// }

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

	// print para análisis
	// fmt.Printf("[DEBUG] Buffer armado: %v\n", buf)
	// fmt.Printf("[DEBUG] Buffer como string: %q\n", buf)

	// sleep para poder inspeccionarlo
	// time.Sleep(5 * time.Second)

	return buf
}

func sendBytesToConnection(connection net.Conn, bytes []byte) int {
	totalSent := 0
	for totalSent < len(bytes) {
		// fmt.Printf("[DEBUG] Enviando: %q\n", string(bytes[totalSent:]))
		n, err := connection.Write(bytes[totalSent:])
		if err != nil {
			// fmt.Println("[ERROR] Error enviando bet al servidor:", err)
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
		bets, finished := getBatchBetFromCSV(datasetPath, batchSize, reader)
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

func receiveWinners(connection net.Conn) (int, error) {
	time.Sleep(5 * time.Second)
	header := make([]byte, 3) // 1 byte tipo + 2 bytes length

	_, err := io.ReadFull(connection, header)
	if err != nil {
		return 0, fmt.Errorf("error leyendo header: %w", err)
	}

	msgType := header[0]
	if msgType != 3 {
		return 0, fmt.Errorf("tipo de mensaje inesperado: %d", msgType)
	}

	msgLen := int(header[1])<<8 | int(header[2])

	msg := make([]byte, msgLen)
	_, err = io.ReadFull(connection, msg)
	if err != nil {
		return 0, fmt.Errorf("error leyendo payload: %w", err)
	}
	payloadStr := string(msg)
	winnerStrs := strings.Split(payloadStr, ",")

	return len(winnerStrs), nil
}
