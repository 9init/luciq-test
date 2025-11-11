package model

import "time"

type Chat struct {
	ID            uint      `db:"id"`
	ApplicationID uint      `db:"application_id"`
	Number        int       `db:"number"`
	MessagesCount int       `db:"messages_count"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}
