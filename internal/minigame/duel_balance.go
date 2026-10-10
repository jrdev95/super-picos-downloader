package minigame

// Experimental balancing depends only on the current winning streak.
// Weights are normalized against the opponent; they are not percentages.
func duelWeight(streak int64) int {
	switch {
	case streak >= 7:
		return 55
	case streak >= 5:
		return 70
	case streak >= 3:
		return 85
	default:
		return 100
	}
}
