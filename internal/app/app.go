package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/deicon/funwithflags/internal/database"
	"github.com/deicon/funwithflags/internal/flag"
	"github.com/deicon/funwithflags/internal/httpserver"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	server *http.Server
	pool   *pgxpool.Pool
}

func New() (*App, error) {
	ctx := context.Background()

	// Determine storage type from environment
	storageType := getEnv("STORAGE_TYPE", "postgres")

	var repo flag.Repository
	var auditService flag.AuditService
	var pool *pgxpool.Pool

	switch storageType {
	case "postgres":
		dbCfg := database.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "funwithflags"),
		}

		// Run migrations
		migrationsPath := getEnv("MIGRATIONS_PATH", "./migrations")
		if err := database.RunMigrations(dbCfg, migrationsPath); err != nil {
			log.Printf("warning: migrations failed: %v", err)
		}

		// Create connection pool
		var err error
		pool, err = database.NewPool(ctx, dbCfg)
		if err != nil {
			return nil, fmt.Errorf("init database pool: %w", err)
		}

		repo = flag.NewPostgresRepository(pool)
		auditService = flag.NewPostgresAuditService(pool)
		log.Println("Using PostgreSQL storage")

	case "memory":
		repo = flag.NewInMemoryRepository()
		log.Println("Using in-memory storage")

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}

	engine := flag.NewEngine()
	flagService, err := flag.NewService(repo, engine)
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, fmt.Errorf("init flag service: %w", err)
	}

	if auditService != nil {
		flagService.SetAuditService(auditService)
	}

	router, err := httpserver.NewRouter(httpserver.Config{
		FlagService: flagService,
	})
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, fmt.Errorf("init router: %w", err)
	}

	port := getEnv("PORT", "8080")
	return &App{
		server: &http.Server{
			Addr:              ":" + port,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
		pool: pool,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func (a *App) Run(ctx context.Context) error {
	if a.server == nil {
		return fmt.Errorf("http server not configured")
	}

	// Cleanup pool on shutdown
	if a.pool != nil {
		defer a.pool.Close()
	}

	listener, err := net.Listen("tcp", a.server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", a.server.Addr, err)
	}
	defer listener.Close()

	log.Printf("Server listening on %s", a.server.Addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		log.Println("Server shutdown complete")
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("serve: %w", err)
	}
}
