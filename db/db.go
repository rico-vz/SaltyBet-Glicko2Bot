package db

import (
	"database/sql"

	"sort"

	"github.com/charmbracelet/log"
	_ "modernc.org/sqlite"
)

var database *sql.DB

type Character struct {
	Name       string
	WinCount   int
	LossCount  int
	Rating     float64
	RD         float64
	Volatility float64
}

func InitializeDB(filepath string) {
	var err error

	database, err = sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatal("Error opening database, ", err)
	}

	createCharactersTableSQL := `
    CREATE TABLE IF NOT EXISTS characters (
        name TEXT NOT NULL PRIMARY KEY UNIQUE,
        win_count INTEGER DEFAULT 0,
        loss_count INTEGER DEFAULT 0,
        rating REAL DEFAULT 1500,
        rd REAL DEFAULT 200,
        volatility REAL DEFAULT 0.06
    );`

	_, err = database.Exec(createCharactersTableSQL)
	if err != nil {
		log.Fatal("Error creating characters table, ", err)
	}
}

func CloseDB() {
	err := database.Close()
	if err != nil {
		log.Fatal("Error closing database")
	}
}

func GetCharacter(name string) (*Character, error) {
	row := database.QueryRow("SELECT name, win_count, loss_count, rating, rd, volatility FROM characters WHERE name = ?", name)

	var character Character
	err := row.Scan(&character.Name, &character.WinCount, &character.LossCount, &character.Rating, &character.RD, &character.Volatility)
	if err != nil {
		return nil, err
	}

	return &character, nil
}

func SaveCharacter(character *Character) error {
	_, err := database.Exec(`
        INSERT OR IGNORE INTO characters (name, win_count, loss_count, rating, rd, volatility) 
        VALUES (?, 0, 0, 1500, 200, 0.06)`,
		character.Name)
	if err != nil {
		log.Printf("Error inserting character: %s, error: %v", character.Name, err)
		return err
	}

	_, err = database.Exec(`
        UPDATE characters 
        SET win_count = ?, loss_count = ?, rating = ?, rd = ?, volatility = ? 
        WHERE name = ?`,
		character.WinCount, character.LossCount, character.Rating, character.RD, character.Volatility, character.Name)
	if err != nil {
		log.Printf("Error updating character: %s, error: %v", character.Name, err)
		return err
	}

	return nil
}

type byRatingDesc []Character

func (a byRatingDesc) Len() int           { return len(a) }
func (a byRatingDesc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byRatingDesc) Less(i, j int) bool { return a[i].Rating > a[j].Rating }

func GetAllCharacters() ([]Character, error) {
	rows, err := database.Query("SELECT name, win_count, loss_count, rating, rd, volatility FROM characters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var characters []Character
	for rows.Next() {
		var character Character
		err := rows.Scan(&character.Name, &character.WinCount, &character.LossCount, &character.Rating, &character.RD, &character.Volatility)
		if err != nil {
			return nil, err
		}
		characters = append(characters, character)
	}

	// Sort characters by rating in descending order before returning
	sort.Sort(byRatingDesc(characters))

	return characters, nil
}
