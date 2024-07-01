package bot

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"rico-vz/SaltyBet-Glicko2Bot/db"
	"rico-vz/SaltyBet-Glicko2Bot/glicko"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gocolly/colly/v2"
	"golang.org/x/exp/rand"
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

type BetState struct {
	BetAmount    int
	ChosenNumber string
}

type BetResult struct {
	Timestamp time.Time `json:"timestamp"`
	BetAmount int       `json:"betAmount"`
	Chosen    string    `json:"chosen"`
	Result    string    `json:"result"`
	Balance   string    `json:"balance"`
}

var botState *BotState
var betState *BetState

var phpSessID string
var defaultBetAmount int
var highBetAmount int
var maxBetAmount int

func SetVarFromEnv() {
	phpSessID = os.Getenv("PHPSESSID")
	defaultBetStr := os.Getenv("DEFAULT_BET")
	highBetStr := os.Getenv("HIGH_BET")
	maxBetStr := os.Getenv("MAX_BET")

	defaultBetInt, err := strconv.Atoi(defaultBetStr)
	if err != nil {
		log.Error("Error converting default bet to int: ", "error", err)
		defaultBetInt = 100
	}

	highBetInt, err := strconv.Atoi(highBetStr)
	if err != nil {
		log.Error("Error converting high bet to int: ", "error", err)
		highBetInt = 100
	}

	maxBetInt, err := strconv.Atoi(maxBetStr)
	if err != nil {
		log.Error("Error converting max bet to int: ", "error", err)
		maxBetInt = 100
	}

	defaultBetAmount = defaultBetInt
	highBetAmount = highBetInt
	maxBetAmount = maxBetInt
}

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
	probability := glicko2WinProbability(character1, character2)

	// Determine the higher probability side
	var chosenSide string
	var chosenName string
	if probability == 0.5 {
		// Randomly choose a side if probability is exactly 0.5 (most likely first time we see the characters)
		rand.Seed(uint64(time.Now().UnixNano()))
		if rand.Intn(2) == 0 {
			chosenSide = "player1"
			chosenName = character1.Name
		} else {
			chosenSide = "player2"
			chosenName = character2.Name
		}
	} else if probability > 0.5 {
		chosenSide = "player1"
		chosenName = character1.Name
	} else {
		chosenSide = "player2"
		chosenName = character2.Name
		probability = 1 - probability // Make probability always > 0.5 for our bet amount calculation
	}

	// Adjust bet amount based on probability
	if probability > 0.85 {
		log.Info("High probability, max bet: ", "probability", probability)
		botState.BetAmount = maxBetAmount
	} else if probability > 0.7 {
		log.Info("Medium probability, higher bet: ", "probability", probability)
		rand.Seed(uint64(time.Now().UnixNano()))
		botState.BetAmount = rand.Intn(maxBetAmount-highBetAmount+1) + highBetAmount
	} else if probability > 0.6 {
		log.Info("Meh probability, high bet: ", "probability", probability)
		botState.BetAmount = highBetAmount
	} else {
		botState.BetAmount = defaultBetAmount
	}

	if character1.Rating == 1500 || character2.Rating == 1500 {
		if character1.Rating > 1750 || character2.Rating > 1750 {
			log.Info("One of the characters has >1750 rating, max bet. ", "probability", probability)
			botState.BetAmount = maxBetAmount
		} else {
			log.Info("One of the characters has 1500 rating, default bet. ", "probability", probability)
			botState.BetAmount = defaultBetAmount
		}
	}

	log.Info("Betting 🧂" + strconv.Itoa(botState.BetAmount) + " on " + chosenName + " with a probability of " + fmt.Sprintf("%.2f", probability))
	return chosenSide
}

func glicko2WinProbability(player1, player2 *db.Character) float64 {
	q := math.Ln10 / 400

	g := func(rd float64) float64 {
		return 1 / math.Sqrt(1+3*q*q*rd*rd/math.Pi/math.Pi)
	}

	E := func(r, r_j, RD_j float64) float64 {
		return 1 / (1 + math.Exp(-g(RD_j)*(r-r_j)/400))
	}

	return E(player1.Rating, player2.Rating, player2.RD)
}

func UpdateResults(winner, loser *db.Character) {
	winner.WinCount++
	loser.LossCount++

	glicko.UpdateRatings(winner, loser)

	db.SaveCharacter(winner)
	db.SaveCharacter(loser)
}

func RunBot() {
	SetVarFromEnv()
	ScrapeBalance()

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
			BetAmount:  defaultBetAmount,
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
	// Store the current state in the bot state
	botState = &BotState{
		Character1: &db.Character{Name: state.P1Name, Rating: 1500, RD: 200, Volatility: 0.06},
		Character2: &db.Character{Name: state.P2Name, Rating: 1500, RD: 200, Volatility: 0.06},
		BetAmount:  defaultBetAmount,
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

	chosenCharacter := ChooseCharacter(character1, character2)
	chosenCharacterNum := string(chosenCharacter[len(chosenCharacter)-1])

	betState = &BetState{
		BetAmount:    botState.BetAmount,
		ChosenNumber: chosenCharacterNum,
	}

	// Submit the bet
	SubmitBet(chosenCharacter, botState.BetAmount)

}

func OnMatchEnd(state *SaltyBetState) {
	var winner, loser *db.Character
	var err error

	getCharacter := func(name string) (*db.Character, error) {
		character, err := db.GetCharacter(name)
		if err != nil {
			log.Error("Error getting character from database: ", "error", err)
			return nil, err
		}
		return character, nil
	}

	if state.Status == "1" {
		winner, err = getCharacter(botState.Character1.Name)
		if err != nil {
			return
		}
		loser, err = getCharacter(botState.Character2.Name)
		if err != nil {
			return
		}
	} else {
		winner, err = getCharacter(botState.Character2.Name)
		if err != nil {
			return
		}
		loser, err = getCharacter(botState.Character1.Name)
		if err != nil {
			return
		}
	}

	newBalance := ScrapeBalance()

	if state.Status == betState.ChosenNumber {
		log.Info("Bet won: ", "amount", betState.BetAmount)
		SaveBetResult(BetResult{
			Timestamp: time.Now(),
			BetAmount: betState.BetAmount,
			Chosen:    betState.ChosenNumber,
			Result:    "win",
			Balance:   newBalance,
		})
	} else {
		log.Info("Bet lost: ", "amount", betState.BetAmount)
		SaveBetResult(BetResult{
			Timestamp: time.Now(),
			BetAmount: betState.BetAmount,
			Chosen:    betState.ChosenNumber,
			Result:    "loss",
			Balance:   newBalance,
		})
	}

	UpdateResults(winner, loser)
}

func SubmitBet(player string, amount int) {
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
}

func SaveBetResult(bet BetResult) error {
	if IsTournamentMode() {
		return nil
	}

	var betResults []BetResult
	file, err := os.OpenFile("./bet_results.json", os.O_RDWR|os.O_CREATE, 0644)
	if err == nil {
		defer file.Close()
		json.NewDecoder(file).Decode(&betResults)
	} else {
		betResults = []BetResult{}
	}

	// Append new bet result
	betResults = append(betResults, bet)

	// Write back to file
	file, err = os.Create("./bet_results.json")
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(betResults)
}

func ScrapeBalance() string {
	var balance string

	c := colly.NewCollector()

	c.OnHTML("#balance", func(e *colly.HTMLElement) {
		balance = strings.ReplaceAll(e.Text, ",", "")
		log.Info("Balance: $" + balance)
	})

	c.OnRequest(func(r *colly.Request) {
		log.Info("Visiting: " + r.URL.String())
		r.Headers.Set("Cookie", "PHPSESSID="+phpSessID)
	})

	c.Visit("https://www.saltybet.com/")

	c.Wait()

	return balance
}

func IsTournamentMode() bool {
	c := colly.NewCollector()

	var tournamentMode bool = false

	c.OnHTML("#tournament-note", func(e *colly.HTMLElement) {
		tournamentMode = e.Text == "(Tournament Balance)"
	})

	c.Visit("https://www.saltybet.com/")

	c.Wait()

	return tournamentMode
}
