package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/deicon/funwithflags/internal/flag"
	"github.com/deicon/funwithflags/internal/httpserver"
)

type App struct {
	server *http.Server
}

func New() (*App, error) {
	repo := flag.NewInMemoryRepository()
	engine := flag.NewEngine()
	flagService, err := flag.NewService(repo, engine)
	if err != nil {
		return nil, fmt.Errorf("init flag service: %w", err)
	}

	router, err := httpserver.NewRouter(httpserver.Config{
		FlagService: flagService,
	})
	if err != nil {
		return nil, fmt.Errorf("init router: %w", err)
	}

	return &App{
		server: &http.Server{
			Addr:              ":8080",
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.server == nil {
		return fmt.Errorf("http server not configured")
	}

	listener, err := net.Listen("tcp", a.server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", a.server.Addr, err)
	}
	defer listener.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("serve: %w", err)
	}
}
