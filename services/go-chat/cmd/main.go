package main

import (
	"go-chat/internal/config"
	"go-chat/internal/database"
	"go-chat/internal/elasticsearch"
	"go-chat/internal/logging"
	"go-chat/internal/module/chat"
	"go-chat/internal/queue"
	"go-chat/internal/server"

	"go.uber.org/dig"
)

func buildDigContainer() *dig.Container {
	container := dig.New()
	// Core dependencies
	container.Provide(config.NewConfig)
	container.Provide(logging.NewLogger)
	container.Provide(database.ConnectDatabase)
	container.Provide(queue.NewAMQP)
	container.Provide(elasticsearch.NewClient)

	// Chat dependencies
	container.Provide(chat.NewRepo)
	container.Provide(chat.NewChatService)

	// Server
	container.Provide(server.NewServer)

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

	logger.Info("Starting Go Chat Service")
	err = container.Invoke(func(server *server.Server) {
		server.Start()
	})

	if err != nil {
		logger.Error("Failed to start server: %v", err)
	}
}
