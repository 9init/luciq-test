package model

import "time"

type Message struct {
	ApplicationToken string    `json:"application_token"`
	ApplicationName  string    `json:"application_name"`
	ChatNumber       int       `json:"chat_number"`
	MessageNumber    int       `json:"message_number"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}
