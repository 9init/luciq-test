package chat

import (
	"go-chat/internal/elasticsearch"
	"go-chat/internal/model"
	"go-chat/internal/queue"
)

type Service struct {
	repo  *Repo
	queue *queue.AMQP
	es    *elasticsearch.Client
}

func NewChatService(repo *Repo, amqp *queue.AMQP, es *elasticsearch.Client) *Service {
	return &Service{
		repo:  repo,
		queue: amqp,
		es:    es,
	}
}

func (s *Service) SendMessage(appToken string, chatNumber int, content string) error {
	return s.repo.CreateMessage(appToken, chatNumber, content)
}

func (s *Service) IncrementChatCounter(appToken string) (int64, error) {
	return s.repo.IncrementChatCounter(appToken)
}

func (s *Service) IncrementMessageCounter(appToken string, chatNumber int) (int64, error) {
	return s.repo.IncrementMessageCounter(appToken, chatNumber)
}

func (s *Service) QueueMessage(payload interface{}, queueType queue.QueueType) error {
	return s.queue.PublishMessage(payload, queueType)
}

func (s *Service) SearchMessages(appToken string, chatNumber int, query string, page int, pageSize int) ([]*model.Message, int, error) {
	return s.es.Search(appToken, chatNumber, query, page, pageSize)
}
