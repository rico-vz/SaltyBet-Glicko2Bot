# SaltyBet Glicko-2 Bot

Bot that automatically bets on SaltyBet MUGEN matches and uses the Glicko-2 Rating System to choose. 

Includes a web dashboard to see balance, total bets & bet winrate at a glance.

## Results

Started a fresh new SaltyBet account and the account is currently over 17million balance with >60% winrate.

![SaltyBet Results](https://i.imgur.com/TdJEsk3.png)

The database size is only ~700KB with having >9700 MUGEN characters' data.

![SaltyBet Character Database](https://i.imgur.com/tO0P7s8.png)

## Known Issues

### Code needs to be refactored
- Codebase is quite messy and unorganized. Needs some heavy refactoring.

### Add Glicko-2 formula to pick expected outcome
- Currently expected outcome is picked purely by who has the higher flat Glicko-2 rating. 
- Formula that uses RD & VOL should be implemented to improve winrate further.

### Tournament Balance Bug
- Web dashboard (and raw json) currently treats tournament balance the same as your "real" balance.
- This causes big balance drops on the dashboard stat box & chart. 
