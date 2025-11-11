package database

import (
	"database/sql"
	"fmt"
	"go-worker/internal/model"
	"time"
)

// Repository provides database operations
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindApplicationByToken(token string) (*model.Application, error) {
	var app model.Application
	query := "SELECT id, token, name, chats_count, created_at, updated_at FROM applications WHERE token = ?"

	err := r.db.QueryRow(query, token).Scan(
		&app.ID,
		&app.Token,
		&app.Name,
		&app.ChatsCount,
		&app.CreatedAt,
		&app.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("application not found")
	}
	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (r *Repository) FindChatByApplicationAndNumber(applicationID uint, number int) (*model.Chat, error) {
	var chat model.Chat
	query := "SELECT id, application_id, number, messages_count, created_at, updated_at FROM chats WHERE application_id = ? AND number = ?"

	err := r.db.QueryRow(query, applicationID, number).Scan(
		&chat.ID,
		&chat.ApplicationID,
		&chat.Number,
		&chat.MessagesCount,
		&chat.CreatedAt,
		&chat.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("chat not found")
	}
	if err != nil {
		return nil, err
	}

	return &chat, nil
}

func (r *Repository) FindMessageByChatAndNumber(chatID uint, number int) (*model.Message, error) {
	var message model.Message
	query := "SELECT id, chat_id, number, content, created_at, updated_at FROM messages WHERE chat_id = ? AND number = ?"

	err := r.db.QueryRow(query, chatID, number).Scan(
		&message.ID,
		&message.ChatID,
		&message.Number,
		&message.Content,
		&message.CreatedAt,
		&message.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, err
	}

	return &message, nil
}

func (r *Repository) CreateChat(chat *model.Chat) error {
	query := "INSERT INTO chats (application_id, number, messages_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?)"

	now := time.Now()
	result, err := r.db.Exec(query, chat.ApplicationID, chat.Number, chat.MessagesCount, now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	chat.ID = uint(id)
	chat.CreatedAt = now
	chat.UpdatedAt = now

	return nil
}

func (r *Repository) CreateMessage(message *model.Message) error {
	query := "INSERT INTO messages (chat_id, number, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?)"

	now := time.Now()
	result, err := r.db.Exec(query, message.ChatID, message.Number, message.Content, now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	message.ID = uint(id)
	message.CreatedAt = now
	message.UpdatedAt = now

	return nil
}

func (r *Repository) IncrementApplicationChatCount(appID uint, delta int) error {
	query := "UPDATE applications SET chats_count = chats_count + ? WHERE id = ?"
	_, err := r.db.Exec(query, delta, appID)
	return err
}

func (r *Repository) IncrementChatMessageCount(chatID uint, delta int) error {
	query := "UPDATE chats SET messages_count = messages_count + ? WHERE id = ?"
	_, err := r.db.Exec(query, delta, chatID)
	return err
}
