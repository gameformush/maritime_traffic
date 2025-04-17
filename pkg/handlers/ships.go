package handlers

import (
	"encoding/json"
	"fmt"
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(ships)
}

func (h *ShipsHandler) GetShip(w http.ResponseWriter, r *http.Request) {
	shipID, ok := mux.Vars(r)[muxIDVar]
	if !ok {
		http.Error(w, "ship id can not be empty", http.StatusBadRequest)
		return
	}

	positions, err := h.ships.GetShipPositions(shipID)
	if err != nil { // better error handling
		switch err {
		case traffic.ErrNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	res := GetShipResponse{
		ID:        shipID,
		Positions: positions,
	}

	json.NewEncoder(w).Encode(res)
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

	result, err := h.ships.PositionShip(traffic.PositionShip{
		ID:   shipID,
		Time: req.Time,
		X:    req.X,
		Y:    req.Y,
	})
	if err != nil { // better error handling
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	fmt.Printf("PositionShip: %v\n", req)

	res := PositionShipResponse{
		Time:   req.Time,
		X:      req.X,
		Y:      req.Y,
		Speed:  result.Speed,
		Status: result.Status,
	}

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(res) // TODO handle error
}

func (p PositionShipRequest) Validate() error {
	if p.Time == 0 {
		return fmt.Errorf("time can not be empty")
	}

	if int64(p.Time) > time.Now().Unix() {
		return fmt.Errorf("time can not be in the future")
	}

	return nil
}
