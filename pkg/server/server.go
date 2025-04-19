package server

import (
	"log/slog"
	"maritime_traffic/pkg/handlers"
	"net/http"

	"github.com/gorilla/mux"
)

func NewAPI(shipsH *handlers.ShipsHandler) *mux.Router {
	r := mux.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					slog.Error("panic", "error", err)
				}
			}()
			next.ServeHTTP(w, r)
		})
	})

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slog.Info("request", "method", r.Method, "path", r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	v1 := r.PathPrefix("/api/v1").Subrouter()
	ships := v1.PathPrefix("/ships").Subrouter()
	ships.HandleFunc("", shipsH.GetShips).Methods("GET")
	ships.HandleFunc("/{id}", shipsH.GetShip).Methods("GET")
	ships.HandleFunc("/{id}/position", shipsH.PositionShip).Methods("POST")

	v1.HandleFunc("/flush", shipsH.Flush).Methods("POST")
	return r
}
