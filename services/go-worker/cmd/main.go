package main

import (
	"go-worker/internal/config"
	"go-worker/internal/database"
	"go-worker/internal/elasticsearch"
	"go-worker/internal/logging"
	"go-worker/internal/queue"
	"go-worker/internal/service"
	"go-worker/internal/worker"

	"go.uber.org/dig"
)

func buildDigContainer() *dig.Container {
	container := dig.New()

	container.Provide(config.NewConfig)
	container.Provide(logging.NewLogger)
	container.Provide(database.ConnectDatabase)
	container.Provide(queue.NewAMQP)
	container.Provide(elasticsearch.NewClient)
	container.Provide(queue.NewConsumer)
	container.Provide(worker.NewWorkers)
	container.Provide(service.NewWorkerService)

	return container
}

func main() {
	var logger *logging.Logger
	container := buildDigContainer()

	err := container.Invoke(func(l *logging.Logger) {
		logger = l
	})
	if err != nil {
		panic(err)
	}

	logger.Info("Starting Go Worker Service")

	err = container.Invoke(func(workerService *service.WorkerService) error {
		return workerService.Start()
	})
	if err != nil {
		logger.Error("Failed to start worker service: %v", err)
	}
}
