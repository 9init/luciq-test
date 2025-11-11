package chat

import "github.com/gofiber/fiber/v2"

func (s *Service) GetRouter() *fiber.App {
	route := fiber.New()

	apps := route.Group("/applications")
	apps = apps.Group("/:token")
	apps.Post("/chats", s.CreateChatHandler)
	apps.Post("/chats/:number/messages", s.CreateMessageHandler)
	apps.Get("/chats/:number/messages/search", s.SearchMessagesHandler)

	return route
}
