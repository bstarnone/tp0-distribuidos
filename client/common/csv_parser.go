package common

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

func getBatchBetFromCSV(csvPath string, batchSize int, reader *csv.Reader) ([]Bet, bool) {
	// file, _ := os.Open(csvPath)
	// reader := csv.NewReader(file)
	bets := []Bet{}

	for i := 0; i < batchSize; i++ {
		rawBet, err := reader.Read()
		if err == io.EOF {
			return bets, true
		}
		if err != nil {
			log.Errorf("Error desconocido leyendo el csv")
			return []Bet{}, true
		}
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
	return bets, false
}
