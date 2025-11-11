package server

import (
	"fmt"
	"go-chat/internal/config"
	"go-chat/internal/logging"
	"go-chat/internal/module/chat"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

type Server struct {
	Config      *config.Config
	ChatService *chat.Service
	fiberApp    *fiber.App
	logger      *logging.Logger
}

func (s *Server) Start() error {
	host := s.Config.ListenAddr
	port := s.Config.ListenPort

	return s.fiberApp.Listen(fmt.Sprintf("%s:%d", host, port))
}

func NewServer(cfg *config.Config, logger *logging.Logger, chatService *chat.Service) *Server {
	server := &Server{
		Config:      cfg,
		ChatService: chatService,
		logger:      logger,
		fiberApp:    fiber.New(),
	}

	logger.Info("Initializing Fiber app")
	server.setupServer()
	logger.Info("Fiber app initialized successfully")
	return server
}

func (s *Server) setupServer() {
	s.setupRateLimiter()
	s.setupLogger()
	s.setupRoutes()
}

func (s *Server) setupRoutes() {
	chatRouter := s.ChatService.GetRouter()
	s.fiberApp.Mount("/", chatRouter)
}

func (s *Server) setupRateLimiter() {
	app := s.fiberApp
	app.Use(limiter.New(limiter.Config{
		Max:        10,
		Expiration: 5 * time.Second,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).Send(nil)
		},
	}))
}

func (s *Server) setupLogger() {
	app := s.fiberApp
	app.Get("/metrics", monitor.New())
	app.Use(requestid.New())
	// app.Use(logger.New(
	// 	logger.Config{
	// 		Output: s.logger,
	// 	},
	// ))
	app.Use(func(c *fiber.Ctx) error {
		trace := newTrace(s.Config.AppName)
		reqLogger := s.logger.WithPrefix(trace)
		c.Locals("logger", reqLogger)
		return c.Next()
	})
}

var counter uint64 = 0

func newTrace(appname string) string {
	id := atomic.AddUint64(&counter, 1)
	t := time.Now().UTC()
	return fmt.Sprintf("%s.%08X.%08X", strings.ToUpper(appname), t.Unix(), id)
}
