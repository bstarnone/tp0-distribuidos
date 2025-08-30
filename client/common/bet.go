package common

import "fmt"

type Bet struct {
	Nombre     string
	Apellido   string
	DNI        uint32
	Nacimiento string
	Numero     uint32
}

func (b Bet) Serialize() []byte {
	return []byte(fmt.Sprintf("%s;%s;%d;%s;%d", b.Nombre, b.Apellido, b.DNI, b.Nacimiento, b.Numero))
}
