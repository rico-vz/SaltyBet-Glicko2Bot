# SaltyBet Glicko-2 Bot

Bot that automatically bets on SaltyBet MUGEN matches and uses the Glicko-2 Rating System to choose. 

Includes a web dashboard to see balance, total bets & bet winrate at a glance.

## Known Issues

### Code needs to be refactored
- Codebase is quite messy and unorganized. Needs some heavy refactoring.

### Add Glicko-2 formula to pick expected outcome
- Currently expected outcome is picked purely by who has the higher flat Glicko-2 rating. 
- Formula that uses RD & VOL needs to be implemented.

### Tournament Balance Bug
- Web dashboard (and raw json) currently treats tournament balance the same as your "real" balance.
- This causes big balance drops on the dashboard stat box & chart. 