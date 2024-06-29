package glicko

import (
	"rico-vz/SaltyBet-Glicko2Bot/db"

	"github.com/charmbracelet/log"
	glicko "github.com/zelenin/go-glicko2"
)

func UpdateRatings(winner, loser *db.Character) {
	log.Info("Winner: " + winner.Name + " Loser: " + loser.Name)

	winnerRating := glicko.NewRating(winner.Rating, winner.RD, winner.Volatility)
	loserRating := glicko.NewRating(loser.Rating, loser.RD, loser.Volatility)

	winnerPlayer := glicko.NewPlayer(winnerRating)
	loserPlayer := glicko.NewPlayer(loserRating)

	period := glicko.NewRatingPeriod()
	period.AddMatch(winnerPlayer, loserPlayer, glicko.MATCH_RESULT_WIN)

	period.Calculate()

	winner.Rating = winnerPlayer.Rating().R()
	winner.RD = winnerPlayer.Rating().Rd()
	winner.Volatility = winnerPlayer.Rating().Sigma()

	loser.Rating = loserPlayer.Rating().R()
	loser.RD = loserPlayer.Rating().Rd()
	loser.Volatility = loserPlayer.Rating().Sigma()
}
