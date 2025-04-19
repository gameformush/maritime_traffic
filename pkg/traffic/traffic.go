package traffic

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Status string

const (
	Green  Status = "green"
	Yellow Status = "yellow"
	Red    Status = "red"

	maxSpeedPerSecond = 100.0 // Maximum speed of a ship in units per second

	YellowThreshold = 1.0 // Distance threshold for yellow status
	RedThreshold    = 0   // Distance threshold for red status

	predictionTimeSeconds = 60.0
	epsilon               = 1e-9 // For floating point comparisons
)

type ShipPosition struct {
	Time     int
	Position Vector
	Speed    Vector
}

type Ship struct {
	ID           string  `json:"id"`
	LastSeen     string  `json:"last_time"`
	LastStatus   Status  `json:"last_status"`
	LastSpeed    float64 `json:"last_speed"`
	LastPosition Vector  `json:"last_position"`
}

type PositionShip struct {
	ID    string
	Time  int
	Point Vector
}

type PositionResult struct {
	Speed  float64
	Status Status
}

type Traffic struct {
	mu         sync.RWMutex
	History    map[string][]ShipPosition
	LastStatus map[string]Status
}

var (
	ErrNotFound     = errors.New("ship not found")
	ErrTimeInPast   = errors.New("time must be greater than last position time")
	ErrTimeInFuture = errors.New("time must be in the past")
)

func NewTraffic() *Traffic {
	t := &Traffic{
		History:    make(map[string][]ShipPosition),
		LastStatus: make(map[string]Status),
	}

	return t
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
			ship.LastSpeed = lastPosition.Speed.Magnitude()
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
	if ps.Time > int(time.Now().Unix()) {
		return PositionResult{}, ErrTimeInFuture
	}

	var (
		speed        Vector
		lastPosition ShipPosition
	)

	t.mu.RLock()
	if len(t.History[ps.ID]) > 0 {
		lastPosition = t.History[ps.ID][len(t.History[ps.ID])-1]
	}
	t.mu.RUnlock()

	if lastPosition.Time != 0 {
		if ps.Time <= lastPosition.Time {
			return PositionResult{}, ErrTimeInPast
		}

		deltaTime := float64(ps.Time - lastPosition.Time)
		deltaX := ps.Point.X - lastPosition.Position.X
		deltaY := ps.Point.Y - lastPosition.Position.Y
		speed = Vector{
			X: float64(deltaX) / deltaTime,
			Y: float64(deltaY) / deltaTime,
		}
	}

	status := evaluateTrafficStatus(t, ps, speed)

	t.mu.Lock()
	t.LastStatus[ps.ID] = status
	t.History[ps.ID] = append(t.History[ps.ID], ShipPosition{
		Time:     ps.Time,
		Speed:    speed,
		Position: ps.Point,
	})
	t.mu.Unlock()

	return PositionResult{
		Speed:  speed.Magnitude(),
		Status: status,
	}, nil
}

// evaluateTrafficStatus goes over all ships
// find ship states before ps.Time + 60
// calculate ship position at ps.Time
// calculate distance between the two ships
// if distance > 60 * maxSpeed * 2 then skip they are too far
// otherwise calculate min distance
// if status red then break, not going to get better
// if status yellow then set status yellow
//
// edge cases:
// 0,0 - tower TODO
//
// locks RLock on t.Histor
func evaluateTrafficStatus(t *Traffic, ps PositionShip, speed Vector) Status {
	status := Green

	t.mu.RLock()
	defer t.mu.RUnlock()

	for shipID, history := range t.History {
		if shipID == ps.ID {
			continue // don't collide with itself
		}

		otherShip := rewindShipBinarySearch(history, ps)
		if otherShip.Time == 0 {
			continue // no history for this time
		}

		currentPosition := otherShip.Position
		if otherShip.Time < ps.Time { // estimate position at ps.Time
			currentPosition = otherShip.Position.Add(otherShip.Speed.ScalarMultiply(float64(ps.Time - otherShip.Time)))
		}
		if currentPosition.Subtract(ps.Point).Magnitude() > maxSpeedPerSecond*predictionTimeSeconds*2 {
			continue // no way to be close
		}

		minDist := calculateMinDistance(ShipPosition{
			Position: currentPosition,
			Speed:    otherShip.Speed,
		}, ShipPosition{
			Position: ps.Point,
			Speed:    speed,
		}, predictionTimeSeconds)

		newStatus := statusForDist(minDist)
		if newStatus == Red {
			status = newStatus
			break
		}

		if status != Yellow {
			status = newStatus
		}
	}

	return status
}

func rewindShipBinarySearch(history []ShipPosition, ps PositionShip) ShipPosition {
	idx := sort.Search(len(history), func(i int) bool {
		return history[i].Time > ps.Time+predictionTimeSeconds
	})
	if idx == 0 {
		return ShipPosition{}
	}

	return history[idx-1]
}

func rewindShip(history []ShipPosition, ps PositionShip) ShipPosition {
	ship := ShipPosition{}
	for i := 0; i < len(history); i++ {
		if history[i].Time > ps.Time+predictionTimeSeconds {
			break
		}
		ship = history[i]
	}

	return ship
}

func statusForDist(minDist float64) Status {
	if minDist < 1 {
		return Red
	}
	if minDist < 2 {
		return Yellow
	}

	return Green
}

// calculateMinDistance calculates the minimum distance between two ships
// over a given duration. It uses the relative position and velocity of the
// ships to determine the time of closest approach and computes the distance
// at that time, as well as at the start and end of the duration.
func calculateMinDistance(s1, s2 ShipPosition, duration float64) float64 {
	rPos := s1.Position.Subtract(s2.Position)
	rVel := s1.Speed.Subtract(s2.Speed)

	relSpeedSq := rVel.MagnitudeSquared()

	// If relative speed is zero, distance is constant
	if relSpeedSq < epsilon {
		return rPos.Magnitude()
	}

	dotProduct := rPos.Dot(rVel)
	// Time of closest approach
	tMin := -dotProduct / relSpeedSq

	distAt0 := rPos.MagnitudeSquared()
	distAtDur := distAt(s1, s2, duration)
	distAtTmin := math.MaxFloat64
	if tMin > 0 && tMin < duration {
		distAtTmin = distAt(s1, s2, tMin)
	}

	return math.Sqrt(min(
		distAt0,
		distAtDur,
		distAtTmin,
	))
}

func distAt(s1, s2 ShipPosition, duration float64) float64 {
	tDur1 := s1.Position.Add(s1.Speed.ScalarMultiply(duration))
	tDur2 := s2.Position.Add(s2.Speed.ScalarMultiply(duration))

	return tDur1.Subtract(tDur2).MagnitudeSquared()
}
