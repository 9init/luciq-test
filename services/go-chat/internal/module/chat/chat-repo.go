package chat

import (
	"context"
	"fmt"
	"go-chat/internal/database"

	"github.com/go-redis/redis/v8"
)

type Repo struct {
	redisClient *redis.Client
}

func NewRepo(db *database.Database) *Repo {
	return &Repo{
		redisClient: db.RedisDB,
	}
}

func (r *Repo) IncrementChatCounter(appToken string) (int64, error) {
	key := fmt.Sprintf("app:%s:chats_count", appToken)
	return r.redisClient.Incr(context.Background(), key).Result()
}

func (r *Repo) IncrementMessageCounter(appToken string, chatNumber int) (int64, error) {
	key := fmt.Sprintf("app:%s:chat:%d:messages_count", appToken, chatNumber)
	return r.redisClient.Incr(context.Background(), key).Result()
}

func (r *Repo) CreateMessage(appToken string, chatNumber int, content string) error {
	messageID, err := r.IncrementMessageCounter(appToken, chatNumber)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("app:%s:chat:%d:messages", appToken, chatNumber)
	messageData := map[string]interface{}{
		"id":      messageID,
		"content": content,
	}

	return r.redisClient.HSet(context.Background(), key, fmt.Sprintf("%d", messageID), messageData).Err()
}
