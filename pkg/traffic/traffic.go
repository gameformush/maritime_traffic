package traffic

import (
	"errors"
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
	ID   string
	Time int
	X    int
	Y    int
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
	ErrNotFound = errors.New("ship not found")
)

func NewTraffic() *Traffic {
	return &Traffic{
		History: make(map[string][]ShipPosition),
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
	if len(t.History[ps.ID]) > 1 {
		lastPosition := t.History[ps.ID][len(t.History[ps.ID])-1]
		dist := (ps.X-lastPosition.Position.X)*(ps.X-lastPosition.Position.X) + (ps.Y-lastPosition.Position.Y)*(ps.Y-lastPosition.Position.Y)
		speed = int(dist / (ps.Time - lastPosition.Time))
	}

	t.History[ps.ID] = append(t.History[ps.ID], ShipPosition{
		Time:     ps.Time,
		Speed:    speed,
		Position: Point{X: ps.X, Y: ps.Y},
	})

	status := Green

	t.LastStatus[ps.ID] = status

	return PositionResult{
		Speed:  speed,
		Status: status,
	}, nil
}
