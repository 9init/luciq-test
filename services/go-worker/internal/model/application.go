package model

import "time"

type Application struct {
	ID         uint      `db:"id"`
	Token      string    `db:"token"`
	Name       string    `db:"name"`
	ChatsCount int       `db:"chats_count"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
