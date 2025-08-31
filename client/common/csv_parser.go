package common

import (
	"encoding/csv"
	"fmt"
	"os"
)

func getBetFromCSV(csvPath string) Bet {
	file, _ := os.Open(csvPath)

	reader := csv.NewReader(file)
	rawBet, _ := reader.Read()
	fmt.Println("[RAW BET]", rawBet)

	bet := Bet{
		AgenciaID:  os.Getenv("CLI_ID"),
		Nombre:     rawBet[0],
		Apellido:   rawBet[1],
		DNI:        rawBet[2],
		Nacimiento: rawBet[3],
		Numero:     rawBet[4],
	}

	fmt.Println("[BET]", bet)

	return bet
}
