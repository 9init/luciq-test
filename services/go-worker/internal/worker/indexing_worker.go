package worker

import (
	"encoding/json"
	"fmt"
	"go-worker/internal/elasticsearch"
	"go-worker/internal/logging"
	"go-worker/internal/queue"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type IndexingWorker struct {
	es          *elasticsearch.Client
	logger      *logging.Logger
	batch       []IndexPayload
	batchMutex  sync.Mutex
	batchSize   int
	flushTicker *time.Ticker
	stopChan    chan struct{}
}

type IndexPayload struct {
	MessageID        uint      `json:"message_id"`
	ApplicationID    uint      `json:"application_id"`
	ApplicationToken string    `json:"application_token"`
	ApplicationName  string    `json:"application_name"`
	ChatID           uint      `json:"chat_id"`
	ChatNumber       int       `json:"chat_number"`
	MessageNumber    int       `json:"message_number"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

func NewIndexingWorker(es *elasticsearch.Client, logger *logging.Logger) *IndexingWorker {
	w := &IndexingWorker{
		es:          es,
		logger:      logger.WithPrefix("IndexingWorker"),
		batch:       make([]IndexPayload, 0, 1000),
		batchSize:   1000,
		flushTicker: time.NewTicker(5 * time.Second),
		stopChan:    make(chan struct{}),
	}

	go w.startAutoFlush()

	return w
}

func (w *IndexingWorker) HandleMessage(delivery amqp.Delivery) error {
	var payload IndexPayload
	if err := queue.ParseMessageBody(delivery, &payload); err != nil {
		w.logger.Error("Failed to parse: %v", err)
		return nil
	}

	w.batchMutex.Lock()
	w.batch = append(w.batch, payload)
	shouldFlush := len(w.batch) >= w.batchSize
	w.batchMutex.Unlock()

	if shouldFlush {
		return w.flush()
	}

	return nil
}

func (w *IndexingWorker) flush() error {
	w.batchMutex.Lock()
	if len(w.batch) == 0 {
		w.batchMutex.Unlock()
		return nil
	}

	messages := make([]IndexPayload, len(w.batch))
	copy(messages, w.batch)
	w.batch = w.batch[:0]
	w.batchMutex.Unlock()

	w.logger.Info("Flushing %d messages to Elasticsearch", len(messages))
	var bulkBody string
	for _, msg := range messages {
		// Route by app+chat to keep all messages from same chat on same shard
		routing := fmt.Sprintf("%s:%d", msg.ApplicationToken, msg.ChatNumber)

		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_id": fmt.Sprintf("%s:%d:%d",
					msg.ApplicationToken, msg.ChatNumber, msg.MessageNumber),
				"routing": routing,
			},
		}
		actionJSON, _ := json.Marshal(action)
		bulkBody += string(actionJSON) + "\n"

		doc := map[string]interface{}{
			"application_token": msg.ApplicationToken,
			"application_name":  msg.ApplicationName,
			"chat_number":       msg.ChatNumber,
			"message_number":    msg.MessageNumber,
			"content":           msg.Content,
			"created_at":        msg.CreatedAt,
		}
		docJSON, _ := json.Marshal(doc)
		bulkBody += string(docJSON) + "\n"
	}

	if err := w.es.BulkIndex("messages", bulkBody); err != nil {
		w.logger.Error("Bulk index failed: %v", err)

		w.batchMutex.Lock()
		w.batch = append(messages, w.batch...)
		w.batchMutex.Unlock()

		return err
	}

	w.logger.Info("Successfully indexed %d messages", len(messages))
	w.resetFlushTimer()

	return nil
}

func (w *IndexingWorker) startAutoFlush() {
	for {
		select {
		case <-w.flushTicker.C:
			if err := w.flush(); err != nil {
				w.logger.Error("Auto-flush failed: %v", err)
			}
		case <-w.stopChan:
			return
		}
	}
}

func (w *IndexingWorker) resetFlushTimer() {
	w.flushTicker.Reset(5 * time.Second)
}

func (w *IndexingWorker) Stop() {
	close(w.stopChan)
	w.flushTicker.Stop()

	if err := w.flush(); err != nil {
		w.logger.Error("Final flush failed: %v", err)
	}
}
