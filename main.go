package main

import (
	"encoding/json"
	"path/filepath"
	"rico-vz/SaltyBet-Glicko2Bot/bot"
	"rico-vz/SaltyBet-Glicko2Bot/db"
	_ "rico-vz/SaltyBet-Glicko2Bot/glicko"
	"strings"

	"net/http"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

func envLoad() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func htmlFileHandler(root http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request path ends with "/" or has no extension, assume it's an HTML file
		if strings.HasSuffix(r.URL.Path, "/") || filepath.Ext(r.URL.Path) == "" {
			// Try serving the path with .html extension
			htmlPath := r.URL.Path + ".html"
			if _, err := root.Open(htmlPath); err == nil {
				r.URL.Path = htmlPath
			}
		}

		// Serve the file (or directory) from the root FileSystem
		http.FileServer(root).ServeHTTP(w, r)
	})
}

func startFrontend() {
	log.Info("[FE] Starting Frontend")
	log.Info("[FE] Listening on http://127.0.0.1:5807/")

	http.Handle("/", http.StripPrefix("/", htmlFileHandler(http.Dir("./views"))))
	http.Handle("/bet_results.json", http.FileServer(http.Dir("./")))
	// /api/characters returns db.GetAllCharacters()
	http.HandleFunc("/api/characters", func(w http.ResponseWriter, r *http.Request) {
		characters, err := db.GetAllCharacters()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		charactersBytes, err := json.Marshal(characters)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(charactersBytes)
	})
	http.ListenAndServe(":5807", nil)
}

func main() {
	envLoad()

	db.InitializeDB("db/characters.db")
	defer db.CloseDB()

	go startFrontend()

	bot.RunBot()
}
