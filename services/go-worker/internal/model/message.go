package model

import "time"

type Message struct {
	ID        uint      `db:"id"`
	ChatID    uint      `db:"chat_id"`
	Number    int       `db:"number"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
