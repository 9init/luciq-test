package chat

import (
	"go-chat/internal/logging"
	"go-chat/internal/model"
	"go-chat/internal/queue"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func (s *Service) CreateChatHandler(ctx *fiber.Ctx) error {
	logger := ctx.Locals("logger").(*logging.Logger)
	appToken := ctx.Params("token")

	if appToken == "" {
		logger.Error("missing app token in URL")
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "app token is required",
		})
	}

	logger.Info("creating chat for app: %s", appToken)
	chatNumber, err := s.IncrementChatCounter(appToken)
	if err != nil {
		logger.Error("failed to increment chat counter: %v", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to increment chat counter",
		})
	}
	logger.Info("chat number generated: %d for app %s", chatNumber, appToken)

	payload := map[string]interface{}{
		"app_token":   appToken,
		"chat_number": chatNumber,
	}
	if err := s.QueueMessage(payload, queue.ChatsQueue); err != nil {
		logger.Error("failed to queue chat for persistence: %v", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to queue chat",
		})
	}
	logger.Info("chat queued successfully: number=%d", chatNumber)

	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"number": chatNumber,
		"status": "processing",
	})
}

func (s *Service) CreateMessageHandler(ctx *fiber.Ctx) error {
	logger := ctx.Locals("logger").(*logging.Logger)
	appToken := ctx.Params("token")
	chatNumberStr := ctx.Params("number")

	if appToken == "" {
		logger.Error("missing app token in URL")
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "app token is required",
		})
	}

	if chatNumberStr == "" {
		logger.Error("missing chat number in URL")
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "chat number is required",
		})
	}

	chatNumber, err := strconv.Atoi(chatNumberStr)
	if err != nil {
		logger.Error("invalid chat number: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "chat number must be a valid integer",
		})
	}

	var input model.Message
	if err := ctx.BodyParser(&input); err != nil {
		logger.Error("failed to parse request body: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if input.Content == "" {
		logger.Error("missing content in request body")
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "content is required",
		})
	}

	messageNumber, err := s.IncrementMessageCounter(appToken, chatNumber)
	if err != nil {
		logger.Error("failed to increment message counter: %v", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to increment message counter",
		})
	}
	logger.Info("message number generated: %d for chat %d", messageNumber, chatNumber)

	payload := map[string]interface{}{
		"app_token":      appToken,
		"chat_number":    chatNumber,
		"content":        input.Content,
		"message_number": messageNumber,
	}

	if err := s.QueueMessage(payload, queue.MessagesQueue); err != nil {
		logger.Error("failed to queue message for persistence: %v", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to queue message",
		})
	}
	logger.Info("message queued successfully for chat %d", chatNumber)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"number": messageNumber,
	})
}

func (s *Service) SearchMessagesHandler(ctx *fiber.Ctx) error {
	logger := ctx.Locals("logger").(*logging.Logger)
	appToken := ctx.Params("token")
	chatNumberStr := ctx.Params("number")
	query := ctx.Query("q")
	pageStr := ctx.Query("page", "1")
	perPageStr := ctx.Query("per_page", "20")

	if appToken == "" {
		logger.Error("missing app token in URL")
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "app token is required",
		})
	}

	if chatNumberStr == "" {
		logger.Error("missing chat number in URL")
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "chat number is required",
		})
	}

	if query == "" {
		logger.Error("missing search query")
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "query parameter 'q' is required",
		})
	}

	chatNumber, err := strconv.Atoi(chatNumberStr)
	if err != nil {
		logger.Error("invalid chat number: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "chat number must be a valid integer",
		})
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(perPageStr)
	if err != nil || perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	logger.Info("searching messages: app=%s, chat=%d, query=%s, page=%d, per_page=%d", appToken, chatNumber, query, page, perPage)

	messages, total, err := s.SearchMessages(appToken, chatNumber, query, page, perPage)
	if err != nil {
		logger.Error("failed to search messages: %v", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to search messages",
		})
	}

	totalPages := (total + perPage - 1) / perPage

	logger.Info("found %d messages matching query (total: %d)", len(messages), total)

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": messages,
		"meta": fiber.Map{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}
