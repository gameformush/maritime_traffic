package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maritime_traffic/pkg/traffic"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const muxIDVar = "id"

type (
	IShips interface {
		GetShips() ([]traffic.Ship, error)
		GetShipPositions(id string) ([]traffic.ShipPosition, error)
		PositionShip(ps traffic.PositionShip) (traffic.PositionResult, error)
		Flush()
	}
	ShipsHandler struct {
		ships IShips
	}
	PositionShipRequest struct {
		Time int `json:"time"`
		X    int `json:"x"`
		Y    int `json:"y"`
	}
	PositionShipResponse struct {
		Time   int            `json:"time"`
		X      int            `json:"x"`
		Y      int            `json:"y"`
		Speed  int            `json:"speed"`
		Status traffic.Status `json:"status"`
	}
	GetShipResponse struct {
		ID        string                 `json:"id"`
		Positions []traffic.ShipPosition `json:"positions"`
	}
)

func NewShipsHandler(ships IShips) *ShipsHandler {
	return &ShipsHandler{
		ships: ships,
	}
}

func (h *ShipsHandler) Flush(w http.ResponseWriter, r *http.Request) {
	h.ships.Flush()

	w.WriteHeader(http.StatusNoContent)
}

func (h *ShipsHandler) GetShips(w http.ResponseWriter, r *http.Request) {
	ships, err := h.ships.GetShips()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	sendJSON(w, ships)
}

func (h *ShipsHandler) GetShip(w http.ResponseWriter, r *http.Request) {
	shipID, ok := mux.Vars(r)[muxIDVar]
	if !ok {
		http.Error(w, "ship id can not be empty", http.StatusBadRequest)
		return
	}

	positions, err := h.ships.GetShipPositions(shipID)
	if err != nil {
		switch err {
		case traffic.ErrNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	sendJSON(w, GetShipResponse{
		ID:        shipID,
		Positions: positions,
	})
}

func (h *ShipsHandler) PositionShip(w http.ResponseWriter, r *http.Request) {
	shipID, ok := mux.Vars(r)[muxIDVar]
	if !ok {
		http.Error(w, "ship id can not be empty", http.StatusBadRequest)
		return
	}

	var req PositionShipRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.ships.PositionShip(traffic.PositionShip{
		ID:   shipID,
		Time: req.Time,
		Point: traffic.Point{
			X: req.X,
			Y: req.Y,
		},
	})
	if err != nil {
		switch err {
		case traffic.ErrTimeInPast:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)

	sendJSON(w, PositionShipResponse{
		Time:   req.Time,
		X:      req.X,
		Y:      req.Y,
		Speed:  result.Speed,
		Status: result.Status,
	})
}

func (p PositionShipRequest) Validate() error {
	if p.Time == 0 {
		return fmt.Errorf("time can not be empty")
	}

	if p.Time < 0 {
		return fmt.Errorf("time can not be negative")
	}

	if int64(p.Time) > time.Now().Unix() {
		return fmt.Errorf("time can not be in the future")
	}

	return nil
}

func sendJSON(w http.ResponseWriter, message any) {
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(message)
	if err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
