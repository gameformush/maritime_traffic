package server

import (
	"maritime_traffic/pkg/handlers"

	"github.com/gorilla/mux"
)

func NewAPI(shipsH *handlers.ShipsHandler) *mux.Router {
	r := mux.NewRouter()

	v1 := r.PathPrefix("/api/v1").Subrouter()
	ships := v1.PathPrefix("/ships").Subrouter()
	ships.HandleFunc("", shipsH.GetShips).Methods("GET")
	ships.HandleFunc("/{id}", shipsH.GetShip).Methods("GET")
	ships.HandleFunc("/{id}/position", shipsH.PositionShip).Methods("POST")

	v1.HandleFunc("/flush", shipsH.Flush).Methods("POST")
	return r
}
