package main

import (
	"rico-vz/SaltyBet-Glicko2Bot/bot"
	"rico-vz/SaltyBet-Glicko2Bot/db"
	_ "rico-vz/SaltyBet-Glicko2Bot/glicko"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

func envLoad() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	envLoad()

	db.InitializeDB("db/characters.db")
	defer db.CloseDB()

	bot.RunBot()

}
