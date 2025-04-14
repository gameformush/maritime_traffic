package server

import (
	"fmt"
	"log/slog"
	"maritime_traffic/pkg/handlers"
	"maritime_traffic/pkg/server"
	"maritime_traffic/pkg/traffic"
	"net/http"

	"github.com/gorilla/mux"
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
	flushH := handlers.NewFlushHandler()

	r := mux.NewRouter()

	v1 := r.PathPrefix("/api/v1").Subrouter()

	ships := v1.PathPrefix("/ships").Subrouter()
	ships.HandleFunc("", shipsH.GetShips).Methods("GET")
	ships.HandleFunc("/{id}", shipsH.GetShip).Methods("GET")
	ships.HandleFunc("/{id}/position", shipsH.PositionShip).Methods("POST")

	v1.HandleFunc("/flush", flushH.Flush).Methods("POST")

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
