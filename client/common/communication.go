package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

type BetMessage struct {
	data []byte
	len  uint32
}

func (b Bet) SerializeBet() []byte {
	return []byte(fmt.Sprintf("%d;%s;%s;%d;%s;%d", b.AgenciaID, b.Nombre, b.Apellido, b.DNI, b.Nacimiento, b.Numero))
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
