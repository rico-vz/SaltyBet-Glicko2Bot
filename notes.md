# Glicko2Bot Notes

### [GET] https://www.saltybet.com/state.json?t={UNIX_TIMESTAMP}

**Example Response:**
```json
{
  "p1name": "{PLAYER1_NAME}",
  "p2name": "{PLAYER2_NAME}",
  "p1total": "{BETS_TOTAL_ON_P1}",
  "p2total": "{BETS_TOTAL_ON_P1}",
  "status": "{STATUS}",
  "alert": "", // Unknown, hasn't been empty yet
  "x": 1, // Seems to be 1 if betting is closed, 0 if still open
  "remaining": "{MATCHES_LEFT} more matches until the next tournament!"
}
```

- **STATUS**: open, locked, 1 or 2
- ^- **1** = p1name Won & **2** = p2name Won

**Notes:**

Poll every 2 seconds?

### [POST] https://www.saltybet.com/ajax_place_bet.php

**Cookies**: PHPSESSID (Probably required)

**form-data:**
```json
selectedplayer={PLAYER_TO_WIN}&wager=100
```

**Notes:**

Only use when status is open