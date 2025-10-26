package app

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/deicon/funwithflags/internal/httpserver"
)

func TestNew(t *testing.T) {
	a, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.server == nil {
		t.Fatal("expected server to be configured")
	}
}

func TestRun_ShutdownOnContextCancel(t *testing.T) {
	a := &App{
		server: &http.Server{
			Addr:    "127.0.0.1:0",
			Handler: httpserver.NewRouter(),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Run(ctx)
	}()

	select {
	case <-time.After(100 * time.Millisecond):
		// allow server to start before signaling shutdown
	case err := <-errCh:
		t.Fatalf("run returned early: %v", err)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not return after cancel")
	}
}
