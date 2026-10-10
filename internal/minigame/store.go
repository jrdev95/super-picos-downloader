package minigame

import (
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"

	_ "modernc.org/sqlite"
)

// Store serializes local transactions; SQLite also locks writers across processes.
type Store struct {
	db         *sql.DB
	location   *time.Location
	now        func() time.Time
	random     func(int) int
	duelRandom func(int) (int, error)
}
type Player struct {
	ID                                                    int64
	Name                                                  string
	Size, Debt                                            int64
	LastGrow                                              string
	LastGrowAt                                            time.Time
	GrowStreak, BestGrow, Duels, Wins, WinStreak, BestWin int64
	Won, Lost                                             int64
}
type Result struct {
	Text   string
	DuelID int64
}
type RuleError string

func (e RuleError) Error() string { return string(e) }

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	loc, err := time.LoadLocation("America/Fortaleza")
	if err != nil {
		db.Close()
		return nil, err
	}
	s := &Store{db: db, location: loc, now: time.Now, random: rand.IntN, duelRandom: randomDuelSide}
	_, err = db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS game_lock (id INTEGER PRIMARY KEY, value INTEGER NOT NULL);
 INSERT OR IGNORE INTO game_lock VALUES (1,0);
 CREATE TABLE IF NOT EXISTS game_players (chat INTEGER NOT NULL, user INTEGER NOT NULL, data TEXT NOT NULL, PRIMARY KEY(chat,user));
 CREATE TABLE IF NOT EXISTS game_grows (chat INTEGER NOT NULL,user INTEGER NOT NULL,day TEXT NOT NULL,gross INTEGER NOT NULL,paid INTEGER NOT NULL,PRIMARY KEY(chat,user,day));
 CREATE TABLE IF NOT EXISTS game_duels (id INTEGER PRIMARY KEY AUTOINCREMENT,chat INTEGER NOT NULL,creator INTEGER NOT NULL,amount INTEGER NOT NULL CHECK(amount>0),done INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE IF NOT EXISTS game_usernames (chat INTEGER NOT NULL,user INTEGER NOT NULL,username TEXT NOT NULL,PRIMARY KEY(chat,user),UNIQUE(chat,username));
 `)
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }

const growInterval = 6 * time.Hour

func randomDuelSide(total int) (int, error) {
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return 0, fmt.Errorf("falha ao sortear duelo: %w", err)
	}
	return int(n.Int64()), nil
}

func growWait(next, now time.Time) string {
	minutes := int((next.Sub(now) + time.Minute - 1) / time.Minute)
	return fmt.Sprintf("⏳ Próxima tentativa em %dh %dm.", minutes/60, minutes%60)
}
func (s *Store) transaction(fn func(*sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE game_lock SET value=0 WHERE id=1`); err != nil {
		return err
	}
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
func read(tx *sql.Tx, chat, id int64) (Player, error) {
	p := Player{ID: id}
	var data string
	err := tx.QueryRow(`SELECT data FROM game_players WHERE chat=? AND user=?`, chat, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	err = json.Unmarshal([]byte(data), &p)
	return p, err
}
func save(tx *sql.Tx, chat int64, p Player) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO game_players(chat,user,data) VALUES(?,?,?) ON CONFLICT(chat,user) DO UPDATE SET data=excluded.data`, chat, p.ID, string(data))
	return err
}
func ranking(tx *sql.Tx, chat int64) ([]Player, error) {
	rows, err := tx.Query(`SELECT data FROM game_players WHERE chat=? ORDER BY CAST(json_extract(data,'$.Size') AS INTEGER) DESC,user ASC`, chat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Player
	for rows.Next() {
		var data string
		var p Player
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(data), &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func position(players []Player, id int64) int {
	for i, p := range players {
		if p.ID == id {
			return i + 1
		}
	}
	return 0
}
func rate(p Player) int64 {
	if p.Duels == 0 {
		return 0
	}
	return int64(float64(p.Wins)*100/float64(p.Duels) + 0.5)
}
func (s *Store) roll() int64 {
	weights := []int{2, 2, 2, 10, 10, 10, 10, 10, 10, 3, 3, 3, 3, 3, 1, 1, 1}
	n := s.random(84)
	for i, w := range weights {
		if n < w {
			return int64(i)
		}
		n -= w
	}
	panic("invalid random result")
}
func cleanName(name string, id int64) string {
	name = strings.Join(strings.Fields(name), " ")
	r := []rune(name)
	if len(r) > 100 {
		name = string(r[:100])
	}
	if name == "" {
		return fmt.Sprintf("Jogador %d", id)
	}
	return name
}
func (s *Store) Command(chat, id int64, name, command, args string) (result Result, err error) {
	err = s.transaction(func(tx *sql.Tx) error {
		now := s.now().In(s.location)
		day := now.Format("2006-01-02")
		p, e := read(tx, chat, id)
		if e != nil {
			return e
		}
		p.Name = cleanName(name, id)
		if e = save(tx, chat, p); e != nil {
			return e
		}
		switch command {
		case "grow":
			if !p.LastGrowAt.IsZero() && now.Before(p.LastGrowAt.Add(growInterval)) {
				return RuleError("🍆 Seu pal ainda está descansando. " + growWait(p.LastGrowAt.Add(growInterval), now))
			}
			wait := growWait(now.Add(growInterval), now)
			gross := s.roll()
			paid := int64(0)
			if p.Debt > 0 {
				paid = min(int64(s.random(3)+1), p.Debt, gross)
			}
			p.Size += gross - paid
			p.Debt -= paid
			if p.LastGrow == day {
				// Multiple attempts on the same day do not inflate daily streaks.
			} else if p.LastGrow == now.AddDate(0, 0, -1).Format("2006-01-02") {
				p.GrowStreak++
			} else {
				p.GrowStreak = 1
			}
			p.BestGrow = max(p.BestGrow, p.GrowStreak)
			p.LastGrow = day
			p.LastGrowAt = now.UTC()
			// Keep the existing schema and legacy daily records; new keys store
			// full timestamps so several attempts per day can be recorded.
			if _, e = tx.Exec(`INSERT INTO game_grows VALUES(?,?,?,?,?)`, chat, id, p.LastGrowAt.Format(time.RFC3339Nano), gross, paid); e != nil {
				return e
			}
			if e = save(tx, chat, p); e != nil {
				return e
			}
			players, e := ranking(tx, chat)
			if e != nil {
				return e
			}
			result.Text = fmt.Sprintf("🍆 Seu pal cresceu %d cm! Agora ele tem %d cm.\n🏆 Posição no rank: %dº\n%s", gross-paid, p.Size, position(players, id), wait)
			if paid > 0 {
				result.Text += fmt.Sprintf("\n🏦 %d cm foram para o banco. Dívida: %d cm.", paid, p.Debt)
				if p.Debt == 0 {
					result.Text += " Empréstimo quitado!"
				}
			}
		case "emprestimo":
			if p.LastGrow == "" {
				return RuleError("Calma! Primeiro tente fazer o amiguinho crescer. Só quem está com 0 cm pode pedir empréstimo.")
			}
			if p.Debt > 0 {
				return RuleError(fmt.Sprintf("O banco recusou seu pedido. Seu pal ainda está pagando a dívida. Dívida atual: %d cm. Quite o empréstimo antes de pedir outro.", p.Debt))
			}
			if p.Size != 0 {
				return RuleError("Só quem está com 0 cm pode pedir empréstimo.")
			}
			p.Size = s.roll()
			p.Debt = p.Size
			if e = save(tx, chat, p); e != nil {
				return e
			}
			result.Text = fmt.Sprintf("O banco liberou %d cm. Dívida atual: %d cm.", p.Size, p.Debt)
			if p.Size == 0 {
				result.Text = "Saiu 0 cm. O banco te deixou na mão, mas sem dívida."
			}
		case "duelo":
			var amount int64
			// Parse strictly: no decimals, trailing tokens or overflow.
			if len(args) == 0 {
				return RuleError("Use /duelo valor (inteiro, mínimo 1 cm).")
			}
			for _, c := range args {
				if c < '0' || c > '9' {
					return RuleError("Use /duelo valor (inteiro, mínimo 1 cm).")
				}
			}
			if _, e = fmt.Sscan(args, &amount); e != nil || amount < 1 {
				return RuleError("Use /duelo valor (inteiro, mínimo 1 cm).")
			}
			if p.Size < amount {
				return RuleError("🍆 Seu pal não cobre essa aposta.")
			}
			inserted, e := tx.Exec(`INSERT INTO game_duels(chat,creator,amount) VALUES(?,?,?)`, chat, id, amount)
			if e != nil {
				return e
			}
			result.DuelID, e = inserted.LastInsertId()
			if e != nil {
				return e
			}
			result.Text = fmt.Sprintf("⚔️ %s apostou %d cm. Quem encara a treta?", p.Name, amount)
		case "rank":
			players, e := ranking(tx, chat)
			if e != nil {
				return e
			}
			var out strings.Builder
			for i, q := range players {
				mark := ""
				if q.LastGrow != "" && q.LastGrow < now.AddDate(0, 0, -7).Format("2006-01-02") {
					mark = " [~]"
				} else if q.LastGrowAt.IsZero() || !now.Before(q.LastGrowAt.Add(growInterval)) {
					mark = " [+]"
				}
				fmt.Fprintf(&out, "%d | %s -- %d cm%s\n", i+1, q.Name, q.Size, mark)
			}
			out.WriteString("\n[+] Já pode tentar crescer novamente.\n[~] Não cresceu o pal há mais de 7 dias.")
			result.Text = out.String()
		case "status":
			players, e := ranking(tx, chat)
			if e != nil {
				return e
			}
			streak := p.GrowStreak
			if p.LastGrow < now.AddDate(0, 0, -1).Format("2006-01-02") {
				streak = 0
			}
			result.Text = fmt.Sprintf("📊 Estatísticas de %s\n\n🍆 Tamanho: %d cm\n🏆 Rank: %dº\n📅 Dias seguidos: %d\n🔥 Maior sequência: %d\n\n⚔️ Duelos: %d\n🏆 Vitórias: %d\n📈 Win rate: %d%%\n🔥 Maior sequência de vitórias: %d\n\n📈 Cm ganhos: %d cm\n📉 Cm perdidos: %d cm\n\n🏦 Dívida atual: %d cm", p.Name, p.Size, position(players, id), streak, p.BestGrow, p.Duels, p.Wins, rate(p), p.BestWin, p.Won, p.Lost, p.Debt)
		default:
			return RuleError("Comando desconhecido.")
		}
		return nil
	})
	return
}

// Reset retains the participants and their names, zeroing all group statistics.
func (s *Store) Reset(chat int64) error {
	return s.transaction(func(tx *sql.Tx) error {
		players, e := ranking(tx, chat)
		if e != nil {
			return e
		}
		for _, p := range players {
			if e = save(tx, chat, Player{ID: p.ID, Name: p.Name}); e != nil {
				return e
			}
		}
		for _, table := range []string{"game_grows", "game_duels"} {
			if _, e = tx.Exec("DELETE FROM "+table+" WHERE chat=?", chat); e != nil {
				return e
			}
		}
		return nil
	})
}
func (s *Store) Accept(chat, duel, id int64, name string) (text string, err error) {
	err = s.transaction(func(tx *sql.Tx) error {
		var creator, amount int64
		var done int
		e := tx.QueryRow(`SELECT creator,amount,done FROM game_duels WHERE id=? AND chat=?`, duel, chat).Scan(&creator, &amount, &done)
		if errors.Is(e, sql.ErrNoRows) || e == nil && done != 0 {
			return RuleError("Essa treta já acabou ou foi apagada.")
		}
		if e != nil {
			return e
		}
		if creator == id {
			return RuleError("Não vale duelar com o próprio pal.")
		}
		a, e := read(tx, chat, creator)
		if e != nil {
			return e
		}
		b, e := read(tx, chat, id)
		if e != nil {
			return e
		}
		b.Name = cleanName(name, id)
		if a.Size < amount || b.Size < amount {
			return RuleError("Os dois precisam ter saldo para essa aposta.")
		}
		winner, loser := a, b
		weightA, weightB := duelWeight(a.WinStreak), duelWeight(b.WinStreak)
		draw, e := s.duelRandom(weightA + weightB)
		if e != nil {
			return e
		}
		if draw < 0 || draw >= weightA+weightB {
			return errors.New("resultado inválido no sorteio do duelo")
		}
		if draw >= weightA {
			winner, loser = b, a
		}
		winner.Size += amount
		loser.Size -= amount
		winner.Duels++
		loser.Duels++
		winner.Wins++
		winner.WinStreak++
		winner.BestWin = max(winner.BestWin, winner.WinStreak)
		loser.WinStreak = 0
		winner.Won += amount
		loser.Lost += amount
		if e = save(tx, chat, winner); e != nil {
			return e
		}
		if e = save(tx, chat, loser); e != nil {
			return e
		}
		if _, e = tx.Exec(`UPDATE game_duels SET done=1 WHERE id=?`, duel); e != nil {
			return e
		}
		players, e := ranking(tx, chat)
		if e != nil {
			return e
		}
		text = fmt.Sprintf("⚔️ DU-E-LO!\n\n🏆 %s levou a melhor.\n🍆 %d cm agora.\n\n💀 %s perdeu %d cm.\n🍆 Restaram %d cm.\n\n📊 Rank após a treta\n%s → %dº\n%s → %dº\n\n🔥 Vencedor\nWin rate: %d%%\nStreak: %d\nMelhor streak: %d\n\n☠️ Perdedor\nWin rate: %d%%", winner.Name, winner.Size, loser.Name, amount, loser.Size, winner.Name, position(players, winner.ID), loser.Name, position(players, loser.ID), rate(winner), winner.WinStreak, winner.BestWin, rate(loser))
		if weightA != 100 || weightB != 100 {
			text += fmt.Sprintf("\n\n🧪 Chances antes do duelo: %s %.1f%% | %s %.1f%%. Ajuste por vitórias consecutivas.", a.Name, 100*float64(weightA)/float64(weightA+weightB), b.Name, 100*float64(weightB)/float64(weightA+weightB))
		}
		return nil
	})
	return
}
