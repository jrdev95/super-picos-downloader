package minigame

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCooldownCrossesMidnightAndSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 10, 23, 30, 0, 0, s.location)
	s.now = func() time.Time { return now }
	s.random = func(int) int { return 0 }
	command(t, s, 1, 1, "grow", "")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.now = func() time.Time { return now }
	now = now.Add(30 * time.Minute)
	if _, err := s.Command(1, 1, "A", "grow", ""); err == nil || !strings.Contains(err.Error(), "5h 30m") {
		t.Fatalf("midnight cooldown: %v", err)
	}
	// Cooldowns remain isolated by group and player.
	command(t, s, 2, 1, "grow", "")
	command(t, s, 1, 2, "grow", "")
	now = now.Add(5*time.Hour + 30*time.Minute)
	command(t, s, 1, 1, "grow", "")
	if get(t, s, 1, 1).GrowStreak != 2 {
		t.Fatal("daily streak lost")
	}
	if err := s.Reset(1); err != nil {
		t.Fatal(err)
	}
	command(t, s, 1, 1, "grow", "")
}

func TestLegacyPlayersAndHistoryArePreserved(t *testing.T) {
	s := testStore(t)
	seed(t, s, 1, Player{ID: 1, Name: "Legacy", Size: 30, Debt: 2, LastGrow: "2026-09-30", GrowStreak: 4, BestGrow: 4, Wins: 3})
	if _, err := s.db.Exec(`INSERT INTO game_grows VALUES(1,1,'2026-09-30',6,0)`); err != nil {
		t.Fatal(err)
	}
	command(t, s, 1, 1, "grow", "")
	p := get(t, s, 1, 1)
	if p.Size != 35 || p.Debt != 1 || p.Wins != 3 || p.GrowStreak != 4 || p.LastGrowAt.IsZero() {
		t.Fatal(p)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM game_grows`).Scan(&count); err != nil || count != 2 {
		t.Fatal(count, err)
	}
	if _, err := s.Command(1, 1, "A", "grow", ""); err == nil {
		t.Fatal("legacy cooldown not started")
	}
}

func TestDuelBothSidesWinRegardlessOfSizeOrHistory(t *testing.T) {
	for _, sizes := range [][2]int64{{10, 1000}, {1000, 10}, {50, 50}} {
		for side := 0; side < 2; side++ {
			s := testStore(t)
			seed(t, s, 1, Player{ID: 1, Size: sizes[0], Wins: 20, WinStreak: 20})
			seed(t, s, 1, Player{ID: 2, Size: sizes[1]})
			s.random = func(int) int { t.Fatal("duel used growth RNG"); return 0 }
			s.duelRandom = func(n int) (int, error) {
				if side == 0 {
					return 0, nil
				}
				return n - 1, nil
			}
			d := command(t, s, 1, 1, "duelo", "1")
			if _, err := s.Accept(1, d.DuelID, 2, "B"); err != nil {
				t.Fatal(err)
			}
			winnerID := int64(side + 1)
			if p := get(t, s, 1, winnerID); p.Size != sizes[side]+1 {
				t.Fatal(p)
			}
		}
	}
}

func TestDuelRandomFailureRollsBack(t *testing.T) {
	s := testStore(t)
	seed(t, s, 1, Player{ID: 1, Size: 10})
	seed(t, s, 1, Player{ID: 2, Size: 20})
	d := command(t, s, 1, 1, "duelo", "2")
	s.duelRandom = func(int) (int, error) { return 0, errors.New("entropy failed") }
	if _, err := s.Accept(1, d.DuelID, 2, "B"); err == nil {
		t.Fatal("ignored random failure")
	}
	if get(t, s, 1, 1).Size != 10 || get(t, s, 1, 2).Size != 20 {
		t.Fatal("balances changed")
	}
	var done int
	if err := s.transaction(func(tx *sql.Tx) error {
		return tx.QueryRow(`SELECT done FROM game_duels WHERE id=?`, d.DuelID).Scan(&done)
	}); err != nil || done != 0 {
		t.Fatal(done, err)
	}
	s.duelRandom = func(n int) (int, error) { return n - 1, nil }
	if _, err := s.Accept(1, d.DuelID, 2, "B"); err != nil {
		t.Fatal(err)
	}
}
