package e2e

import (
	"context"
	"fmt"
	"log/slog"
	"maritime_traffic/pkg/handlers"
	"maritime_traffic/pkg/server"
	"maritime_traffic/pkg/traffic"
	"net/http"
	"os"
	"testing"
)

const port = 7070
const addr = "http://localhost"

func TestMain(m *testing.M) {
	t := traffic.NewTraffic()

	server.NewServer(server.Config{
		Port: port,
	})

	r := server.NewAPI(handlers.NewShipsHandler(t))

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	go func() {
		slog.Info("listening on port", "port", port)
		err := server.ListenAndServe()
		if err != nil {
			if err != http.ErrServerClosed {
				slog.Error("server error", "error", err)
			}
		}
	}()

	exitCode := m.Run()

	server.Shutdown(context.Background())

	os.Exit(exitCode)
}
