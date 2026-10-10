package minigame

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
)

// RememberUsername stores only identity metadata, never a message's contents.
func (s *Store) RememberUsername(chat, id int64, username string) error {
	username = strings.ToLower(username)
	return s.transaction(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM game_usernames WHERE chat=? AND (user=? OR (username=? AND username<>''))`, chat, id, username); err != nil {
			return err
		}
		if username == "" {
			return nil
		}
		_, err := tx.Exec(`INSERT INTO game_usernames VALUES(?,?,?)`, chat, id, username)
		return err
	})
}

func (s *Store) FindUsername(chat int64, username string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT user FROM game_usernames WHERE chat=? AND username=?`, chat, strings.ToLower(strings.TrimPrefix(username, "@"))).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, RuleError("Não conheço esse @ neste grupo. Peça ao destinatário para usar /status e tente novamente.")
	}
	return id, err
}

// Donate changes only balances, preserving debt, cooldowns and duel statistics.
func (s *Store) Donate(chat, from, to, amount int64, fromName, toName string) (text string, err error) {
	err = s.transaction(func(tx *sql.Tx) error {
		if amount < 1 {
			return RuleError("Use /doar quantidade @usuário (inteiro, mínimo 1 cm).")
		}
		if from == to {
			return RuleError("Você não pode doar para si mesmo.")
		}
		a, e := read(tx, chat, from)
		if e != nil {
			return e
		}
		b, e := read(tx, chat, to)
		if e != nil {
			return e
		}
		if a.Size < amount {
			return RuleError("Você não tem centímetros suficientes para essa doação.")
		}
		if b.Size > math.MaxInt64-amount {
			return RuleError("Essa doação excede o saldo permitido.")
		}
		a.Name, b.Name = cleanName(fromName, from), cleanName(toName, to)
		a.Size -= amount
		b.Size += amount
		if e = save(tx, chat, a); e != nil {
			return e
		}
		if e = save(tx, chat, b); e != nil {
			return e
		}
		text = fmt.Sprintf("🍆 %s doou %d cm para %s.\n%s: %d cm\n%s: %d cm", a.Name, amount, b.Name, a.Name, a.Size, b.Name, b.Size)
		return nil
	})
	return
}
