package traffic

import (
	"errors"
	"math"
	"strconv"
	"sync"
)

type Status string

const (
	Green  Status = "green"
	Yellow Status = "yellow"
	Red    Status = "red"
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ShipPosition struct {
	Time     int   `json:"time"`
	Speed    int   `json:"speed"`
	Position Point `json:"position"`
}

type Ship struct {
	ID           string `json:"id"`
	LastSeen     string `json:"last_time"`
	LastStatus   Status `json:"last_status"`
	LastSpeed    int    `json:"last_speed"`
	LastPosition Point  `json:"last_position"`
}

type PositionShip struct {
	ID    string
	Time  int
	Point Point
}

type PositionResult struct {
	Speed  int
	Status Status
}

type Traffic struct {
	mu         sync.RWMutex
	History    map[string][]ShipPosition
	LastStatus map[string]Status
}

var (
	ErrNotFound   = errors.New("ship not found")
	ErrTimeInPast = errors.New("time should be greater than last position time")
)

func NewTraffic() *Traffic {
	return &Traffic{
		History:    make(map[string][]ShipPosition),
		LastStatus: make(map[string]Status),
	}
}

func (t *Traffic) Flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.History = make(map[string][]ShipPosition)
	t.LastStatus = make(map[string]Status)
}

func (t *Traffic) GetShips() ([]Ship, error) {
	res := make([]Ship, 0, len(t.History))

	t.mu.RLock()
	defer t.mu.RUnlock()
	for id, positions := range t.History {
		ship := Ship{
			ID:         id,
			LastStatus: t.LastStatus[id],
		}

		if len(positions) > 0 {
			lastPosition := positions[len(positions)-1]

			ship.LastSeen = strconv.Itoa(lastPosition.Time)
			ship.LastSpeed = lastPosition.Speed
			ship.LastPosition = lastPosition.Position
		}

		res = append(res, ship)
	}

	return res, nil
}

func (t *Traffic) GetShipPositions(id string) ([]ShipPosition, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ship, ok := t.History[id]
	if !ok {
		return nil, ErrNotFound
	}

	return ship, nil
}

func (t *Traffic) PositionShip(ps PositionShip) (PositionResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	speed := 0
	if len(t.History[ps.ID]) > 0 {
		lastPosition := t.History[ps.ID][len(t.History[ps.ID])-1]
		if ps.Time <= lastPosition.Time {
			return PositionResult{}, ErrTimeInPast
		}

		speed = int(distance(lastPosition.Position, ps.Point) / (ps.Time - lastPosition.Time))
	}

	t.History[ps.ID] = append(t.History[ps.ID], ShipPosition{
		Time:     ps.Time,
		Speed:    speed,
		Position: ps.Point,
	})

	// TODO main logic for status
	status := Green

	t.LastStatus[ps.ID] = status

	return PositionResult{
		Speed:  speed,
		Status: status,
	}, nil
}

func distance(p1, p2 Point) int {
	return int(math.Sqrt(float64((p1.X-p2.X)*(p1.X-p2.X) + (p1.Y-p2.Y)*(p1.Y-p2.Y))))
}
