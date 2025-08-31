package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
)

type BetMessage struct {
	data []byte
	len  uint32
}

type BetResponse struct {
	DNI string
	Num string
}

func (b Bet) SerializeBet() []byte {
	return []byte(fmt.Sprintf("%s;%s;%s;%s;%s;%s", b.AgenciaID, b.Nombre, b.Apellido, b.DNI, b.Nacimiento, b.Numero))
}

func uploadBet(connection net.Conn, bet Bet) {
	data := bet.SerializeBet()
	msg := BetMessage{
		data: data,
		len:  uint32(len(data)),
	}
	// TODO: cambiar la lógica para recibir un BetMessage en vez de un Bet

	msg_buf := new(bytes.Buffer)
	binary.Write(msg_buf, binary.BigEndian, msg.len)
	msg_buf.Write(msg.data)

	full_msg := msg_buf.Bytes()
	total_sent := 0

	for total_sent < len(full_msg) {
		n, err := connection.Write(full_msg[total_sent:])
		if err != nil {
			fmt.Println("[ERROR] Error enviando bet al servidor:", err)
			return
		}
		total_sent += n
	}
}

func receiveResponse(connection net.Conn) (*BetResponse, error) {
	lenBuf := make([]byte, 4)
	_, err := io.ReadFull(connection, lenBuf)

	if err != nil {
		return nil, fmt.Errorf("error leyendo header: %w", err)
	}

	responseLen := binary.BigEndian.Uint32(lenBuf)

	msg := make([]byte, responseLen)
	_, err = io.ReadFull(connection, msg)
	if err != nil {
		return nil, fmt.Errorf("error leyendo mensaje: %w", err)
	}

	parts := strings.SplitN(string(msg), ";", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("respuesta inválida: %s", string(msg))
	}

	response := &BetResponse{
		DNI: parts[0],
		Num: parts[1],
	}

	return response, nil
}
