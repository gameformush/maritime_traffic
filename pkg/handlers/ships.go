package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maritime_traffic/pkg/traffic"
	"net/http"

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
	ShipResponse struct {
		ID           string   `json:"id"`
		LastSeen     string   `json:"last_time"`
		LastStatus   string   `json:"last_status"`
		LastSpeed    int      `json:"last_speed"`
		LastPosition Position `json:"last_position"`
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
	Position struct {
		X int `json:"x"`
		Y int `json:"y"`
	}
	ShipPosition struct {
		Time     int      `json:"time"`
		Speed    int      `json:"speed"`
		Position Position `json:"position"`
	}
	GetShipResponse struct {
		ID        string         `json:"id"`
		Positions []ShipPosition `json:"positions"`
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
	sendJSON(w, mapShips(ships))
}

func mapShips(ships []traffic.Ship) []ShipResponse {
	result := make([]ShipResponse, len(ships))
	for i, ship := range ships {
		result[i] = ShipResponse{
			ID:           ship.ID,
			LastSeen:     ship.LastSeen,
			LastStatus:   string(ship.LastStatus),
			LastSpeed:    int(ship.LastSpeed),
			LastPosition: Position{X: int(ship.LastPosition.X), Y: int(ship.LastPosition.Y)},
		}
	}

	return result
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
		Positions: mapPositions(positions),
	})
}

func mapPositions(positions []traffic.ShipPosition) []ShipPosition {
	result := make([]ShipPosition, len(positions))
	for i, pos := range positions {
		result[i] = ShipPosition{
			Time:     pos.Time,
			Speed:    int(pos.Speed.Magnitude()),
			Position: Position{X: int(pos.Position.X), Y: int(pos.Position.Y)},
		}
	}

	return result
}

func (h *ShipsHandler) PositionShip(w http.ResponseWriter, r *http.Request) {
	shipID, ok := mux.Vars(r)[muxIDVar]
	if !ok {
		http.Error(w, "ship id must be provided", http.StatusBadRequest)
		return
	}
	if shipID == "" {
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
		Point: traffic.Vector{
			X: float64(req.X),
			Y: float64(req.Y),
		},
	})
	if err != nil {
		switch err {
		case traffic.ErrTimeInPast, traffic.ErrTimeInFuture:
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
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
		Speed:  int(result.Speed),
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

	return nil
}

func sendJSON(w http.ResponseWriter, message any) {
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(message)
	if err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
