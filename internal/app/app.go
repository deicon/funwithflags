package app

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/deicon/funwithflags/internal/auth"
	"github.com/deicon/funwithflags/internal/database"
	"github.com/deicon/funwithflags/internal/flag"
	"github.com/deicon/funwithflags/internal/httpserver"
	"github.com/deicon/funwithflags/internal/project"
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

	authSecret := getEnv("JWT_SECRET", "funwithflags-dev-secret")
	authAccessTTL := getEnvDuration("AUTH_ACCESS_TOKEN_TTL", 0)
	authRefreshTTL := getEnvDuration("AUTH_REFRESH_TOKEN_TTL", 0)

	adminUsername := getEnv("AUTH_ADMIN_USERNAME", "admin")
	adminPassword := getEnv("AUTH_ADMIN_PASSWORD", "admin123")

	defaultUsername := getEnv("AUTH_USER_USERNAME", "user")
	defaultPassword := getEnv("AUTH_USER_PASSWORD", "user123")

	var repo flag.Repository
	var auditService flag.AuditService
	var pool *pgxpool.Pool
	var projectRepo project.Repository
	var authRepo auth.Repository

	switch storageType {
	case "postgres":
		dbCfg := database.Config{
			URL:      getEnv("DATABASE_URL", ""),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "funwithflags"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
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
		projectRepo = project.NewPostgresRepository(pool)
		authRepo = auth.NewPostgresRepository(pool)
		log.Println("Using PostgreSQL storage")

	case "memory":
		repo = flag.NewInMemoryRepository()
		projectRepo = project.NewInMemoryRepository()
		authRepo = auth.NewInMemoryRepository()
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

	authService, err := auth.NewService(authRepo)
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, fmt.Errorf("init auth service: %w", err)
	}

	if err := authService.EnsureUser(ctx, adminUsername, adminPassword, auth.RoleAdmin); err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, fmt.Errorf("ensure admin user: %w", err)
	}
	if defaultUsername != "" && defaultPassword != "" && defaultUsername != adminUsername {
		if err := authService.EnsureUser(ctx, defaultUsername, defaultPassword, auth.RoleUser); err != nil {
			if pool != nil {
				pool.Close()
			}
			return nil, fmt.Errorf("ensure default user: %w", err)
		}
	}

	authManager, err := auth.NewManager(auth.Config{
		Secret:          authSecret,
		AccessTokenTTL:  authAccessTTL,
		RefreshTokenTTL: authRefreshTTL,
		Repository:      authRepo,
	})
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, fmt.Errorf("init auth manager: %w", err)
	}

	projectService, err := project.NewService(projectRepo)
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, fmt.Errorf("init project service: %w", err)
	}

	if err := projectService.EnsureProjectAndStage(ctx, "default", "Default Project", "production", "Production"); err != nil {
		log.Printf("warning: unable to ensure default project/stage: %v", err)
	}

	allowedOrigins := getEnvStringSlice("CORS_ALLOWED_ORIGINS", []string{
		"https://funwithflags-frontend.fly.dev",
		"http://localhost:5173",
		"http://localhost:3000",
	})

	router, err := httpserver.NewRouter(httpserver.Config{
		FlagService:    flagService,
		ProjectService: projectService,
		AuthManager:    authManager,
		AuthService:    authService,
		AllowedOrigins: allowedOrigins,
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

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValues []string) []string {
	if value, ok := os.LookupEnv(key); ok {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			trimmed = strings.TrimSuffix(trimmed, "/")
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	copyDefaults := make([]string, len(defaultValues))
	copy(copyDefaults, defaultValues)
	return copyDefaults
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
