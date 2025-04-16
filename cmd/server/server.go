package server

import (
	"fmt"
	"log/slog"
	"maritime_traffic/pkg/handlers"
	"maritime_traffic/pkg/server"
	"maritime_traffic/pkg/traffic"
	"net/http"

	"github.com/sethvargo/go-envconfig"
	"github.com/spf13/cobra"
)

func NewServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "start a server",
		Run:   serve,
	}
}

type Config struct {
	Port int `env:"PORT,default=8080"`
}

func serve(cmd *cobra.Command, args []string) {
	t := traffic.NewTraffic()

	var cfg Config
	if err := envconfig.Process(cmd.Context(), &cfg); err != nil {
		slog.Error("failed to load config", "error", err)
		return
	}

	server.NewServer(server.Config{
		Port: cfg.Port,
	})

	shipsH := handlers.NewShipsHandler(t)

	r := server.NewAPI(shipsH)

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: r,
	}

	slog.Info("listening on port", "port", cfg.Port)
	err := server.ListenAndServe()
	if err != nil {
		panic(err) // handle better
	}
}
