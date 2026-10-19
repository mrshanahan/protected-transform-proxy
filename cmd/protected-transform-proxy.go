package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func main() {
	exitCode := Run()
	os.Exit(exitCode)
}

const (
	DefaultPort int    = 3030
	PortEnvVar  string = "PROTECTED_TRANSFORM_PROXY_PORT"
)

func Run() int {
	portStr := os.Getenv(PortEnvVar)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = DefaultPort
		slog.Info("no valid port provided via environment, using default",
			"var", PortEnvVar,
			"portStr", portStr,
			"defaultPort", port)
	} else {
		slog.Info("using custom port",
			"port", port)
	}

	allowedOrigins := "*" // TODO: Change me!
	app := fiber.New()
	app.Use(requestid.New(), logger.New(), recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
	}))
	app.Route("/proxy", func(proxy fiber.Router) {
		proxy.Get("/:id", func(c *fiber.Ctx) error {
			id := c.Params("id", "")
			return c.SendString(fmt.Sprintf("OK %s", id))
		})
	})

	slog.Info("listening for requests", "port", port)
	err = app.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		slog.Error("failed to initialize HTTP server",
			"err", err)
		return 1
	}
	return 0
}
