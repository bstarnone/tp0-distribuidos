package common

import (
	"encoding/csv"
	"fmt"
	"os"
)

func getBatchBetFromCSV(csvPath string, batchSize int) []Bet {
	file, _ := os.Open(csvPath)
	reader := csv.NewReader(file)
	bets := []Bet{}

	for i := 0; i < batchSize; i++ {
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
		bets = append(bets, bet)
	}

	fmt.Println("[CSV BETS Batch]", bets)
	return bets
}
