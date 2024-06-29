package bot

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"rico-vz/SaltyBet-Glicko2Bot/db"
	"rico-vz/SaltyBet-Glicko2Bot/glicko"
	"time"

	"github.com/charmbracelet/log"
)

type SaltyBetState struct {
	P1Name    string `json:"p1name"`
	P2Name    string `json:"p2name"`
	P1Total   string `json:"p1total"`
	P2Total   string `json:"p2total"`
	Status    string `json:"status"`
	Alert     string `json:"alert"`
	X         int    `json:"x"`
	Remaining string `json:"remaining"`
}

var prevState *SaltyBetState = nil

type BotState struct {
	Character1 *db.Character
	Character2 *db.Character
	BetAmount  int
}

var botState *BotState

func fetchSaltyBetState() (*SaltyBetState, error) {
	currentUnixTimestamp := time.Now().Unix()
	resp, err := http.Get(fmt.Sprintf("https://www.saltybet.com/state.json?t=%d", currentUnixTimestamp))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var state SaltyBetState
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func hashState(state *SaltyBetState) string {
	jsonState, _ := json.Marshal(state)
	hash := sha256.Sum256(jsonState)
	return fmt.Sprintf("%x", hash)
}

func ChooseCharacter(character1, character2 *db.Character) string {
	if character1.Rating != character2.Rating {
		if character1.Rating > character2.Rating {
			log.Info("Chosen character: " + character1.Name)
			return "player1"
		} else {
			log.Info("Chosen character: " + character2.Name)
			return "player2"
		}
	}

	if len(character1.Name) > len(character2.Name) {
		log.Info("Chosen character: " + character1.Name)
		return "player1"
	}

	log.Info("Chosen character: " + character2.Name)
	return "player2"
}

func UpdateResults(winner, loser *db.Character) {
	winner.WinCount++
	loser.LossCount++

	glicko.UpdateRatings(winner, loser)

	db.SaveCharacter(winner)
	db.SaveCharacter(loser)
}

func RunBot() {
	var previousHash string

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		state, err := fetchSaltyBetState()
		if err != nil {
			log.Error("Error fetching state: ", "error", err)
			continue
		}

		currentHash := hashState(state)
		if currentHash != previousHash {
			log.Info("State has changed")
			previousHash = currentHash
			// Event
			OnStateChange(state)
		}
	}
}

func OnStateChange(state *SaltyBetState) {
	// If prevState is nil, this is the first state received so we can't compare the status
	// Just store the current state in the prevState for the next call and also botState
	if prevState == nil {
		prevState = state
		botState = &BotState{
			Character1: &db.Character{Name: state.P1Name, Rating: 1500, RD: 200, Volatility: 0.06},
			Character2: &db.Character{Name: state.P2Name, Rating: 1500, RD: 200, Volatility: 0.06},
			BetAmount:  100,
		}
		OnStatusOpened(state)
		return
	}

	if prevState != nil && prevState.Status != "open" && state.Status == "open" {
		OnStatusOpened(state)
	}

	// Check for match end condition (status transition from 'locked' to '1' or '2')
	if prevState != nil && prevState.Status == "locked" && (state.Status == "1" || state.Status == "2") {
		OnMatchEnd(state)
	}

	// Update prevState for the next call
	prevState = state
}

func OnStatusOpened(state *SaltyBetState) {
	// Logic to execute when the status transitions to 'open'
	// Store the current state in the bot state
	botState = &BotState{
		Character1: &db.Character{Name: state.P1Name, Rating: 1500, RD: 200, Volatility: 0.06},
		Character2: &db.Character{Name: state.P2Name, Rating: 1500, RD: 200, Volatility: 0.06},
		BetAmount:  100,
	}

	// Load the characters from the database
	character1, err := db.GetCharacter(state.P1Name)
	if err != nil {
		character1 = botState.Character1
		db.SaveCharacter(character1)
	}

	character2, err := db.GetCharacter(state.P2Name)
	if err != nil {
		character2 = botState.Character2
		db.SaveCharacter(character2)
	}

	// Choose a character to bet on
	chosenCharacter := ChooseCharacter(character1, character2)

	// Submit the bet
	SubmitBet(chosenCharacter, botState.BetAmount)

}

func OnMatchEnd(state *SaltyBetState) {
	var winner, loser *db.Character
	var err error
	if state.Status == "1" {
		winner, err = db.GetCharacter(botState.Character1.Name)
		if err != nil {
			log.Error("Error getting character from database: ", "error", err)
		}

		loser, err = db.GetCharacter(botState.Character2.Name)
		if err != nil {
			log.Error("Error getting character from database: ", "error", err)
		}
	} else {
		winner, err = db.GetCharacter(botState.Character2.Name)
		if err != nil {
			log.Error("Error getting character from database: ", "error", err)
		}

		loser, err = db.GetCharacter(botState.Character1.Name)
		if err != nil {
			log.Error("Error getting character from database: ", "error", err)
		}
	}

	UpdateResults(winner, loser)
}

func SubmitBet(player string, amount int) {
	phpSessID := os.Getenv("PHPSESSID")

	log.Info("Submitting bet: ", "player", player, "amount", amount)

	url := "https://www.saltybet.com/ajax_place_bet.php"
	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("selectedplayer", player)
	_ = writer.WriteField("wager", fmt.Sprintf("%d", amount))
	err := writer.Close()
	if err != nil {
		log.Error("Error closing writer: ", "error", err)
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		log.Error("Error creating request: ", "error", err)
		return
	}
	req.Header.Add("Cookie", "PHPSESSID="+phpSessID+";")

	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		log.Error("Error submitting bet: ", "error", err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error("Error reading response body: ", "error", err)
		return
	}
	log.Info("Bet submitted: ", "response", string(body))
}
