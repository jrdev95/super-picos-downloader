package minigame

import (
	"strings"
	"testing"
)

func TestDuelWeightThresholds(t *testing.T) {
	for _, tc := range []struct {
		streak int64
		weight int
	}{{0, 100}, {2, 100}, {3, 85}, {4, 85}, {5, 70}, {6, 70}, {7, 55}, {100, 55}} {
		if got := duelWeight(tc.streak); got != tc.weight {
			t.Fatalf("streak=%d got=%d", tc.streak, got)
		}
	}
}

func TestBalancedDuelsExactProbabilities(t *testing.T) {
	// Enumerate every possible crypto draw through the real transfer path.
	// No statistical tolerance or flaky random samples are needed.
	for _, tc := range []struct {
		a, b   int64
		wa, wb int
	}{{0, 0, 100, 100}, {3, 0, 85, 100}, {5, 0, 70, 100}, {7, 0, 55, 100}, {7, 7, 55, 55}, {0, 7, 100, 55}} {
		s := testStore(t)
		winsA, winsB := 0, 0
		for ticket := 0; ticket < tc.wa+tc.wb; ticket++ {
			seed(t, s, 1, Player{ID: 1, Name: "A", Size: 1000, WinStreak: tc.a, BestWin: tc.a})
			seed(t, s, 1, Player{ID: 2, Name: "B", Size: 10, WinStreak: tc.b, BestWin: tc.b})
			s.duelRandom = func(n int) (int, error) {
				if n != tc.wa+tc.wb {
					t.Fatalf("wrong draw range %d", n)
				}
				return ticket, nil
			}
			d := command(t, s, 1, 1, "duelo", "1")
			text, err := s.Accept(1, d.DuelID, 2, "B")
			if err != nil {
				t.Fatal(err)
			}
			a, b := get(t, s, 1, 1), get(t, s, 1, 2)
			if a.Size+b.Size != 1010 {
				t.Fatal("centimeters not conserved")
			}
			if a.Size == 1001 {
				winsA++
				if b.WinStreak != 0 || a.WinStreak != tc.a+1 {
					t.Fatal(a, b)
				}
			} else {
				winsB++
				if a.WinStreak != 0 || b.WinStreak != tc.b+1 {
					t.Fatal(a, b)
				}
			}
			if (tc.a >= 3 || tc.b >= 3) != strings.Contains(text, "Chances antes do duelo") {
				t.Fatal("missing or unexpected balancing explanation", text)
			}
		}
		if winsA != tc.wa || winsB != tc.wb {
			t.Fatalf("streaks=%d/%d wins=%d/%d", tc.a, tc.b, winsA, winsB)
		}
	}
}

func TestDefeatRemovesPenalty(t *testing.T) {
	s := testStore(t)
	seed(t, s, 1, Player{ID: 1, Size: 100, WinStreak: 9, BestWin: 9})
	seed(t, s, 1, Player{ID: 2, Size: 100})
	s.duelRandom = func(n int) (int, error) { return n - 1, nil }
	d := command(t, s, 1, 1, "duelo", "1")
	if _, err := s.Accept(1, d.DuelID, 2, "B"); err != nil {
		t.Fatal(err)
	}
	a := get(t, s, 1, 1)
	if duelWeight(a.WinStreak) != 100 || a.BestWin != 9 {
		t.Fatal(a)
	}
	s.duelRandom = func(n int) (int, error) {
		if n != 200 {
			t.Fatalf("penalty remained: %d", n)
		}
		return 0, nil
	}
	d = command(t, s, 1, 1, "duelo", "1")
	if _, err := s.Accept(1, d.DuelID, 2, "B"); err != nil {
		t.Fatal(err)
	}
}
