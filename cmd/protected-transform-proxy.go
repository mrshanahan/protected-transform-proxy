package main

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/mrshanahan/go-utils/auth"
	"github.com/mrshanahan/protected-transform-proxy/internal/routes"
)

func main() {
	exitCode := Run()
	os.Exit(exitCode)
}

const (
	DefaultPort  int    = 3030
	PortEnvVar   string = "PROTECTED_TRANSFORM_PROXY_PORT"
	DbPathEnvVar string = "PROTECTED_TRANSFORM_PROXY_DB_PATH"
)

var (
	DB         *sql.DB
	HttpClient *http.Client

	//go:embed files/create_routes_table.sql
	CreateRoutesTableSql string

	ErrNotExist    error = fmt.Errorf("route with given id does not exist")
	ErrKeyMismatch error = fmt.Errorf("provided key does not match")
)

func initializeDb(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if db == nil {
		panic("WTF???")
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(CreateRoutesTableSql)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return db, nil
}

func getRoute(db *sql.DB, id string, key []byte) (string, error) {
	stmt, err := db.Prepare("SELECT key_hash, key_salt FROM routes WHERE id = ?")
	if err != nil {
		return "", fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	var keyHash, keySalt []byte
	row := stmt.QueryRow(id)
	if err := row.Scan(&keyHash, &keySalt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotExist
		}
		return "", fmt.Errorf("failed to load route: %w", err)
	}

	actualHash, err := auth.Hash(keySalt, key)
	if err != nil {
		return "", fmt.Errorf("failed to hash input: %w", err)
	}
	correct := slices.Equal(actualHash, keyHash)
	if !correct {
		return "", ErrKeyMismatch
	}

	stmt2, err := db.Prepare("SELECT route FROM routes WHERE id = ?")
	if err != nil {
		return "", fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt2.Close()

	var route string
	row2 := stmt2.QueryRow(id)
	if err := row2.Scan(&route); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotExist
		}
		return "", fmt.Errorf("failed to load route: %w", err)
	}

	return route, nil
}

// Production HTTP client with advanced configuration
// Stolen from: https://www.digitalocean.com/community/tutorials/how-to-make-http-requests-in-go#advanced-http-client-configuration
func createProductionClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,              // Maximum idle connections
		MaxIdleConnsPerHost: 10,               // Maximum idle connections per host
		IdleConnTimeout:     90 * time.Second, // Idle connection timeout
		DisableCompression:  false,            // Enable compression
		DisableKeepAlives:   false,            // Enable keep-alives
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Custom redirect handling
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		},
	}
}

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

	dbPath := os.Getenv(DbPathEnvVar)
	if dbPath == "" {
		slog.Error(fmt.Sprintf("FATAL: DB path not provided via %s", DbPathEnvVar))
		return -1
	}
	DB, err = initializeDb(dbPath)
	if err != nil {
		slog.Error("FATAL: failed to initialize database", "err", err, "path", dbPath)
		return -1
	}

	Client := createProductionClient()

	allowedOrigins := "*" // TODO: Change me!
	app := fiber.New()
	app.Use(requestid.New(), logger.New(), recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
	}))
	app.Route("/proxy", func(proxy fiber.Router) {
		proxy.Get("/:id", func(c *fiber.Ctx) error {
			key := c.Query("key", "")
			if key == "" {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
			id := c.Params("id", "")
			route, err := getRoute(DB, id, []byte(key))
			if err != nil {
				if err == ErrNotExist || err == ErrKeyMismatch {
					return c.SendStatus(fiber.StatusNotFound)
				}
				slog.Error("error while attempting to load route", "id", id, "err", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			req, err := http.NewRequestWithContext(c.Context(), "GET", route, nil)
			if err != nil {
				slog.Error("failed to create proxied HTTP request", "err", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			resp, err := Client.Do(req)
			if err != nil {
				slog.Error("failed to send HTTP request", "id", id, "err", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				slog.Error("failed to read response body", "id", id, "err", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}

			c.Set("Content-Type", resp.Header.Get("Content-Type"))
			c.Status(resp.StatusCode)

			transformConfig, prs := routes.Transforms[id]
			if !prs {
				return c.Send(body)
			}
			transformed := transformConfig.Transform(body)
			return c.Send(transformed)
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
