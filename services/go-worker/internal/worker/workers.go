package worker

import (
	"go-worker/internal/database"
	"go-worker/internal/elasticsearch"
	"go-worker/internal/logging"
	"go-worker/internal/queue"
	"sync"
)

// Workers holds all worker instances
type Workers struct {
	Chat           *ChatWorker
	Message        *MessageWorker
	Indexing       *IndexingWorker
	Reconciliation *ReconciliationWorker
}

func NewWorkers(
	db *database.Database,
	amqp *queue.AMQP,
	es *elasticsearch.Client,
	logger *logging.Logger,
) *Workers {
	return &Workers{
		Chat:           NewChatWorker(db, logger),
		Message:        NewMessageWorker(db, amqp, logger),
		Indexing:       NewIndexingWorker(es, logger),
		Reconciliation: NewReconciliationWorker(db, logger),
	}
}

func (w *Workers) Stop() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	if w.Indexing != nil {
		w.Indexing.Stop()
	}
	if w.Reconciliation != nil {
		w.Reconciliation.Stop()
	}
}
