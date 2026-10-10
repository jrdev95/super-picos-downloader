package minigame

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDonationBalancesAndValidation(t *testing.T) {
	s := testStore(t)
	seed(t, s, 1, Player{ID: 1, Size: 10, Debt: 4, LastGrow: "2026-09-30", Wins: 2})
	seed(t, s, 1, Player{ID: 2, Size: 3})
	for _, tc := range []struct{ to, amount int64 }{{1, 1}, {2, 0}, {2, -1}, {2, 11}} {
		if _, err := s.Donate(1, 1, tc.to, tc.amount, "A", "B"); err == nil {
			t.Fatal("invalid donation accepted", tc)
		}
	}
	if _, err := s.Donate(1, 1, 2, 5, "A", "B"); err != nil {
		t.Fatal(err)
	}
	a, b := get(t, s, 1, 1), get(t, s, 1, 2)
	if a.Size != 5 || b.Size != 8 || a.Debt != 4 || a.Wins != 2 || a.LastGrow != "2026-09-30" || b.Duels != 0 {
		t.Fatal(a, b)
	}
	if _, err := s.Donate(2, 1, 2, 1, "A", "B"); err == nil {
		t.Fatal("cross-group balance used")
	}
	seed(t, s, 1, Player{ID: 2, Size: math.MaxInt64})
	if _, err := s.Donate(1, 1, 2, 1, "A", "B"); err == nil {
		t.Fatal("overflow accepted")
	}
	if get(t, s, 1, 1).Size != 5 {
		t.Fatal("failed donation changed balance")
	}
}

func TestConcurrentDonationCannotOverspend(t *testing.T) {
	s := testStore(t)
	seed(t, s, 1, Player{ID: 1, Size: 10})
	var count atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Donate(1, 1, 2, 1, "A", "B"); err == nil {
				count.Add(1)
			}
		}()
	}
	wg.Wait()
	if count.Load() != 10 || get(t, s, 1, 1).Size != 0 || get(t, s, 1, 2).Size != 10 {
		t.Fatal("overspent or lost transfer")
	}
}

func TestUsernameTrackingIsGroupScopedAndHandlesChanges(t *testing.T) {
	s := testStore(t)
	for _, entry := range []struct {
		id   int64
		name string
	}{{1, "Pedro"}, {1, "Novo"}, {2, "PEDRO"}} {
		if err := s.RememberUsername(1, entry.id, entry.name); err != nil {
			t.Fatal(err)
		}
	}
	if id, err := s.FindUsername(1, "@pedro"); err != nil || id != 2 {
		t.Fatal(id, err)
	}
	if _, err := s.FindUsername(2, "@pedro"); err == nil {
		t.Fatal("resolved another group's identity")
	}
	if err := s.RememberUsername(1, 2, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.FindUsername(1, "@pedro"); err == nil {
		t.Fatal("removed username still resolves")
	}
}
