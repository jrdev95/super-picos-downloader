package minigame

import (
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "game.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	s.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, s.location) }
	s.duelRandom = func(int) (int, error) { return 0, nil }
	s.random = func(n int) int {
		if n == 84 {
			return 36
		}
		return 0
	}
	return s
}
func get(t *testing.T, s *Store, chat, id int64) Player {
	t.Helper()
	var p Player
	if e := s.transaction(func(tx *sql.Tx) error { var e error; p, e = read(tx, chat, id); return e }); e != nil {
		t.Fatal(e)
	}
	return p
}
func seed(t *testing.T, s *Store, chat int64, p Player) {
	t.Helper()
	if e := s.transaction(func(tx *sql.Tx) error { return save(tx, chat, p) }); e != nil {
		t.Fatal(e)
	}
}
func command(t *testing.T, s *Store, chat, id int64, cmd, args string) Result {
	t.Helper()
	r, e := s.Command(chat, id, "João", cmd, args)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestGrowConcurrentAndSixHourInterval(t *testing.T) {
	s := testStore(t)
	var success atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := s.Command(1, 1, "João", "grow", ""); e == nil {
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("success=%d", success.Load())
	}
	if p := get(t, s, 1, 1); p.Size != 6 || p.GrowStreak != 1 {
		t.Fatal(p)
	}
	s.now = func() time.Time { return time.Date(2026, 9, 30, 17, 59, 59, 0, s.location) }
	if _, e := s.Command(1, 1, "João", "grow", ""); e == nil {
		t.Fatal("allowed before six hours elapsed")
	}
	s.now = func() time.Time { return time.Date(2026, 9, 30, 18, 0, 0, 0, s.location) }
	command(t, s, 1, 1, "grow", "")
	if p := get(t, s, 1, 1); p.Size != 12 || p.GrowStreak != 1 {
		t.Fatal(p)
	}
	s.now = func() time.Time { return time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC) }
	command(t, s, 1, 1, "grow", "")
	if p := get(t, s, 1, 1); p.Size != 18 || p.GrowStreak != 2 || p.BestGrow != 2 {
		t.Fatal(p)
	}
	s.now = func() time.Time { return time.Date(2026, 10, 3, 3, 0, 0, 0, time.UTC) }
	command(t, s, 1, 1, "grow", "")
	if p := get(t, s, 1, 1); p.GrowStreak != 1 || p.BestGrow != 2 {
		t.Fatal(p)
	}
	var count int
	if e := s.db.QueryRow(`SELECT count(*) FROM game_grows`).Scan(&count); e != nil || count != 4 {
		t.Fatalf("history=%d: %v", count, e)
	}
}
func TestLoanZeroAndRepayment(t *testing.T) {
	s := testStore(t)
	s.random = func(int) int { return 0 }
	s.now = func() time.Time { return time.Date(2026, 9, 29, 12, 0, 0, 0, s.location) }
	command(t, s, 1, 1, "grow", "")
	s.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, s.location) }
	command(t, s, 1, 1, "emprestimo", "")
	if p := get(t, s, 1, 1); p.Debt != 0 || p.Size != 0 {
		t.Fatal(p)
	}
	s.random = func(n int) int {
		if n == 84 {
			return 36
		}
		return 2
	}
	command(t, s, 1, 1, "emprestimo", "")
	if _, e := s.Command(1, 1, "a", "emprestimo", ""); e == nil {
		t.Fatal("second loan allowed")
	}
	s.random = func(n int) int {
		if n == 84 {
			return 2
		}
		return 2
	}
	command(t, s, 1, 1, "grow", "")
	if p := get(t, s, 1, 1); p.Size != 6 || p.Debt != 5 {
		t.Fatal(p)
	}
	s.now = func() time.Time { return time.Date(2026, 10, 1, 12, 0, 0, 0, s.location) }
	s.random = func(int) int { return 0 }
	command(t, s, 1, 1, "grow", "")
	if p := get(t, s, 1, 1); p.Debt != 5 || p.Size != 6 {
		t.Fatal(p)
	}
	seed(t, s, 1, Player{ID: 2, Debt: 2})
	s.random = func(n int) int {
		if n == 84 {
			return 36
		}
		return 2
	}
	r := command(t, s, 1, 2, "grow", "")
	if p := get(t, s, 1, 2); p.Size != 4 || p.Debt != 0 || !strings.Contains(r.Text, "quitado") {
		t.Fatal(p, r)
	}
}
func TestDuelsIndependentAtomicAndValidation(t *testing.T) {
	s := testStore(t)
	seed(t, s, 1, Player{ID: 1, Name: "A", Size: 100})
	seed(t, s, 1, Player{ID: 2, Name: "B", Size: 100})
	for _, arg := range []string{"", "0", "-1", "1.5", "1 2", "999999999999999999999", "101"} {
		if _, e := s.Command(1, 1, "A", "duelo", arg); e == nil {
			t.Fatalf("accepted %q", arg)
		}
	}
	duel := command(t, s, 1, 1, "duelo", "10")
	if _, e := s.Accept(1, duel.DuelID, 1, "A"); e == nil {
		t.Fatal("self duel")
	}
	if _, e := s.Accept(2, duel.DuelID, 2, "B"); e == nil {
		t.Fatal("cross group")
	}
	var count atomic.Int32
	var wg sync.WaitGroup
	for range 15 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := s.Accept(1, duel.DuelID, 2, "B"); e == nil {
				count.Add(1)
			}
		}()
	}
	wg.Wait()
	if count.Load() != 1 {
		t.Fatal(count.Load())
	}
	for range 3 {
		d := command(t, s, 1, 1, "duelo", "10")
		if _, e := s.Accept(1, d.DuelID, 2, "B"); e != nil {
			t.Fatal(e)
		}
	}
	a, b := get(t, s, 1, 1), get(t, s, 1, 2)
	if a.Size != 140 || b.Size != 60 || a.Wins != 4 || a.BestWin != 4 || a.Won != 40 || b.Lost != 40 || b.Duels != 4 {
		t.Fatal(a, b)
	}
	s.random = func(n int) int { return 1 }
	s.duelRandom = func(n int) (int, error) { return n - 1, nil }
	d := command(t, s, 1, 1, "duelo", "10")
	if _, e := s.Accept(1, d.DuelID, 2, "B"); e != nil {
		t.Fatal(e)
	}
	a, b = get(t, s, 1, 1), get(t, s, 1, 2)
	if a.WinStreak != 0 || a.BestWin != 4 || rate(a) != 80 || b.WinStreak != 1 {
		t.Fatal(a, b)
	}
	d = command(t, s, 1, 1, "duelo", "100")
	seed(t, s, 1, Player{ID: 1, Size: 0})
	if _, e := s.Accept(1, d.DuelID, 2, "B"); e == nil {
		t.Fatal("stale balance accepted")
	}
}
func TestResetIsolationPersistenceAndMarkers(t *testing.T) {
	s := testStore(t)
	seed(t, s, 1, Player{ID: 1, Name: "A", Size: 100, LastGrow: "2026-09-22", Debt: 3, Wins: 1, Duels: 2})
	seed(t, s, 1, Player{ID: 2, Name: "B", Size: 100, LastGrow: "2026-09-23"})
	seed(t, s, 2, Player{ID: 1, Name: "A", Size: 99})
	r := command(t, s, 1, 3, "rank", "")
	if !strings.Contains(r.Text, "1 | A -- 100 cm [~]") || !strings.Contains(r.Text, "2 | B -- 100 cm [+]") {
		t.Fatal(r.Text)
	}
	d := command(t, s, 1, 1, "duelo", "1")
	if e := s.Reset(1); e != nil {
		t.Fatal(e)
	}
	p := get(t, s, 1, 1)
	if p.Size != 0 || p.Debt != 0 || p.Duels != 0 || p.Wins != 0 {
		t.Fatal(p)
	}
	if p = get(t, s, 2, 1); p.Size != 99 {
		t.Fatal(p)
	}
	if _, e := s.Accept(1, d.DuelID, 2, "B"); e == nil {
		t.Fatal("reset duel accepted")
	}
	command(t, s, 1, 1, "grow", "")
	path := filepath.Join(t.TempDir(), "persist.db")
	other, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	command(t, other, 8, 9, "status", "")
	if e = other.Close(); e != nil {
		t.Fatal(e)
	}
	other, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	if p = get(t, other, 8, 9); p.Name != "João" {
		t.Fatal(p)
	}
}
func TestDistribution(t *testing.T) {
	s := testStore(t)
	counts := make([]int, 17)
	for i := 0; i < 84; i++ {
		s.random = func(int) int { return i }
		counts[s.roll()]++
	}
	for i, c := range counts {
		expected := 2
		if i >= 3 && i <= 8 {
			expected = 10
		}
		if i >= 9 && i <= 13 {
			expected = 3
		}
		if i >= 14 {
			expected = 1
		}
		if c != expected {
			t.Fatalf("value=%d count=%d", i, c)
		}
	}
}

func TestConcurrentStoresAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")
	a, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	now := func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }
	a.now = now
	b.now = now
	a.random = func(int) int { return 36 }
	b.random = a.random
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		s := a
		if i%2 == 0 {
			s = b
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := s.Command(1, 1, "Persistido", "grow", ""); e == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("multiple connections allowed %d grows", successes.Load())
	}
	p := get(t, b, 1, 1)
	if p.Size != 6 || p.LastGrow != "2026-09-30" || p.BestGrow != 1 {
		t.Fatal(p)
	}
	c, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.now = now
	if _, e = c.Command(1, 1, "Persistido", "grow", ""); e == nil {
		t.Fatal("restart permitted duplicate grow")
	}
	if p = get(t, c, 1, 1); p.Size != 6 {
		t.Fatal(p)
	}
}

func TestLoanRequiresGrowthInCurrentGroup(t *testing.T) {
	s := testStore(t)
	const message = "Calma! Primeiro tente fazer o amiguinho crescer. Só quem está com 0 cm pode pedir empréstimo."
	blocked := func(chat int64) {
		t.Helper()
		_, err := s.Command(chat, 1, "João", "emprestimo", "")
		if err == nil || err.Error() != message {
			t.Fatalf("expected growth requirement, got %v", err)
		}
		if p := get(t, s, chat, 1); p.Size != 0 || p.Debt != 0 {
			t.Fatal(p)
		}
	}
	blocked(1)
	command(t, s, 1, 1, "status", "")
	blocked(1)
	s.random = func(int) int { return 0 }
	command(t, s, 1, 1, "grow", "")
	blocked(2)
	// A zero on the first growth attempt is enough to qualify.
	s.random = func(int) int { return 36 }
	command(t, s, 1, 1, "emprestimo", "")
	if p := get(t, s, 1, 1); p.Size != 6 || p.Debt != 6 {
		t.Fatal(p)
	}
	if err := s.Reset(1); err != nil {
		t.Fatal(err)
	}
	blocked(1)
	// A player who grows and then loses everything in a duel qualifies too.
	command(t, s, 1, 1, "grow", "")
	if _, err := s.Command(1, 1, "João", "emprestimo", ""); err == nil {
		t.Fatal("loan with positive size")
	}
	command(t, s, 1, 2, "grow", "")
	d := command(t, s, 1, 1, "duelo", "6")
	s.random = func(int) int { return 1 }
	s.duelRandom = func(n int) (int, error) { return n - 1, nil }
	if _, err := s.Accept(1, d.DuelID, 2, "Oponente"); err != nil {
		t.Fatal(err)
	}
	if p := get(t, s, 1, 1); p.Size != 0 {
		t.Fatal(p)
	}
	s.random = func(int) int { return 36 }
	command(t, s, 1, 1, "emprestimo", "")
}
