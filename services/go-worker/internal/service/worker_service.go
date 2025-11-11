package service

import (
	"go-worker/internal/logging"
	"go-worker/internal/queue"
	"go-worker/internal/worker"
	"os"
	"os/signal"
	"syscall"
)

type WorkerService struct {
	consumer *queue.Consumer
	workers  *worker.Workers
	logger   *logging.Logger
}

func NewWorkerService(
	consumer *queue.Consumer,
	workers *worker.Workers,
	logger *logging.Logger,
) *WorkerService {
	return &WorkerService{
		consumer: consumer,
		workers:  workers,
		logger:   logger,
	}
}

func (s *WorkerService) Start() error {
	s.logger.Info("Starting workers...")

	// Start chat worker
	err := s.consumer.ConsumeQueue(
		string(queue.ChatsQueue),
		s.workers.Chat.HandleMessage,
	)
	if err != nil {
		return err
	}
	s.logger.Info("Chat worker started on queue: %s", queue.ChatsQueue)

	// Start message worker
	err = s.consumer.ConsumeQueue(
		string(queue.MessagesQueue),
		s.workers.Message.HandleMessage,
	)
	if err != nil {
		return err
	}
	s.logger.Info("Message worker started on queue: %s", queue.MessagesQueue)

	// Start indexing worker
	err = s.consumer.ConsumeQueue(
		string(queue.IndexingQueue),
		s.workers.Indexing.HandleMessage,
	)
	if err != nil {
		return err
	}
	s.logger.Info("Indexing worker started on queue: %s", queue.IndexingQueue)

	s.logger.Info("All workers started successfully")
	s.logger.Info("Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	s.logger.Info("Shutting down workers gracefully...")
	s.Stop()
	s.logger.Info("Workers stopped")

	return nil
}

func (s *WorkerService) Stop() {
	s.workers.Stop()
	s.consumer.Close()
}
