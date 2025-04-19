package traffic

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Status int

const (
	Green Status = iota
	Yellow
	Red
)

const (
	maxSpeedPerSecond = 100.0 // Maximum speed of a ship in units per second

	YellowThreshold = 2 // Distance threshold for yellow status
	RedThreshold    = 1 // Distance threshold for red status

	predictionTimeSeconds = 60.0
	epsilon               = 1e-9 // For floating point comparisons
)

type (
	ShipPosition struct {
		Time     int
		Position Vector
		Speed    Vector
	}
	Ship struct {
		ID           string  `json:"id"`
		LastSeen     string  `json:"last_time"`
		LastStatus   Status  `json:"last_status"`
		LastSpeed    float64 `json:"last_speed"`
		LastPosition Vector  `json:"last_position"`
	}

	PositionShip struct {
		ID    string
		Time  int
		Point Vector
	}

	PositionResult struct {
		Speed  float64
		Status Status
	}

	Traffic struct {
		mu         sync.RWMutex
		History    map[string][]ShipPosition
		LastStatus map[string]Status
	}
)

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

	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.History[ps.ID]) > 0 {
		lastPosition = t.History[ps.ID][len(t.History[ps.ID])-1]
	}

	if lastPosition.Time != 0 {
		if ps.Time <= lastPosition.Time {
			return PositionResult{}, ErrTimeInPast
		}

		deltaTime := float64(ps.Time - lastPosition.Time)
		speed = calculateShipSpeed(deltaTime, ps.Point, lastPosition.Position)
	}

	status := t.evaluateTrafficStatus(ps, speed)

	t.LastStatus[ps.ID] = status
	t.History[ps.ID] = append(t.History[ps.ID], ShipPosition{
		Time:     ps.Time,
		Speed:    speed,
		Position: ps.Point,
	})

	return PositionResult{
		Speed:  speed.Magnitude(),
		Status: status,
	}, nil
}

// calculateShipSpeed between two positions in deltatime and truncate to maxSpeedPerSecond
func calculateShipSpeed(deltaTime float64, newPosition, lastPosition Vector) Vector {
	deltaX := newPosition.X - lastPosition.X
	deltaY := newPosition.Y - lastPosition.Y
	speed := Vector{
		X: float64(deltaX) / deltaTime,
		Y: float64(deltaY) / deltaTime,
	}

	// truncate speed to maxSpeedPerSecond
	if speed.Magnitude() > maxSpeedPerSecond {
		speed = speed.Normalize().ScalarMultiply(maxSpeedPerSecond)
	}

	return speed
}

// evaluateTrafficStatus goes over all ships
// find ship states before ps.Time + 60
// calculate ship position at ps.Time
// calculate distance between the two ships
// each ship can have many positions whithin 60 seconds window
// we need to find the first ship position that is before ps.Time + 60 or right on it
// move first ship to ps.Time to make things easier
// keep ships in time sync
// calculate distance between the two ships
// if status red then break, not going to get any better
// if status yellow then set status yellow
//
// edge cases:
// 0,0 - tower
// ships can jump surpassing max speed - try to use future position to calculate speed,
// speed may not be correct, but at least trajectory is correct
func (t *Traffic) evaluateTrafficStatus(ps PositionShip, speed Vector) Status {
	status := Green

loop:
	for shipID, history := range t.History {
		if shipID == ps.ID {
			continue // don't collide with itself
		}

		// other ships already aligned into the [ps.Time: ps.Time + 60 window]
		// with adujusted speed(code is prettier now :) )
		// move both ships to ts and calculate distance
		currentPosition := ps.Point
		currentTime := ps.Time
		maxPredictionTime := ps.Time + int(predictionTimeSeconds)
		collisionCandidates := rewindShipBinarySearch(history, ps)
		for i, otherShip := range collisionCandidates {
			if otherShip.Time == 0 {
				continue // no history for this time
			}

			// because there are many updates possible within 60 seconds
			// dist calculation must be done for smaller time windows not just +60
			nextPredictionTime := maxPredictionTime
			if i < len(collisionCandidates)-1 {
				nextPredictionTime = min(collisionCandidates[i+1].Time, maxPredictionTime)
			}

			// ships must be at the time for calculate min distance to work
			currentPosition = currentPosition.Add(speed.ScalarMultiply(float64(otherShip.Time - currentTime)))
			currentTime = otherShip.Time

			minDist := calculateMinDistance(ShipPosition{
				Position: otherShip.Position,
				Speed:    otherShip.Speed,
			}, ShipPosition{
				Position: currentPosition,
				Speed:    speed,
			}, float64(nextPredictionTime-currentTime))

			newStatus := statusForDist(minDist)
			if newStatus == Red {
				status = newStatus
				break loop
			}

			if status != Yellow {
				status = newStatus
			}
		}
	}

	towerStatus := checkTowerCollision(ps, speed)
	if status == Green {
		status = towerStatus
	} else if towerStatus == Yellow && status != Red {
		status = Yellow
	}

	return status
}

func checkTowerCollision(ps PositionShip, speed Vector) Status {
	minDist := calculateMinDistance(ShipPosition{
		Position: Vector{X: 0, Y: 0},
		Speed:    Vector{X: 0, Y: 0},
	}, ShipPosition{
		Position: ps.Point,
		Speed:    speed,
	}, predictionTimeSeconds)

	return statusForDist(minDist)
}

// find time box starting at ps.Time and ending at ps.Time + 60
// maybe second search for the end could be linear? - depends on density of updates
// with small density for next 60 seconds second linear search will be very fast
// however I don't want to make assumptions about the density of updates
// so we will use binary search for both
// on second thought, linear search could have better CPU cache performance - benchmark later
func rewindShipBinarySearch(history []ShipPosition, ps PositionShip) []ShipPosition {
	if len(history) == 0 {
		return nil
	}

	startIndex := sort.Search(len(history), func(i int) bool {
		return history[i].Time >= ps.Time
	})
	// didn't find anything, try to go back one
	if startIndex == len(history) && history[startIndex-1].Time < ps.Time {
		startIndex = startIndex - 1
	} else if startIndex > 0 && history[startIndex].Time > ps.Time { // first is out of range try to go back one
		startIndex = startIndex - 1
	}

	endIndex := sort.Search(len(history), func(i int) bool {
		return history[i].Time > ps.Time+int(predictionTimeSeconds)
	})
	// all out of range
	if startIndex == endIndex && startIndex == len(history) {
		return nil
	}

	candidates := make([]ShipPosition, endIndex-startIndex)
	// copy because they are going to be modified and there will not to many of them, at last 60(or max prediction window)
	copy(candidates, history[startIndex:endIndex])
	if len(candidates) == 0 {
		return nil
	}

	// calc actual speed for all candidates if we have next position
	// if range doesn't cover ps.Time + 60 look ahead one more position
	// endIndex points to the first position that is greater than ps.Time + 60
	lastIndex := endIndex
	if endIndex != len(history) && history[endIndex-1].Time < ps.Time+int(predictionTimeSeconds) {
		lastIndex = endIndex + 1
	}

	speedCandidates := history[startIndex:lastIndex]
	for i := range speedCandidates {
		if i < len(speedCandidates)-1 {
			// Why is it needed?
			// At time of ps.Time we might not see future positions which will tell us
			// real trajectory of the ship
			// e.g.
			// time = 1, x = 0, y = 0, speed = 0,0
			// time = 2, x = 1, y = 1, speed = 1,1
			// time = 100 x = 100, y = 0, speed = 1,0 -- very different trajectory from the last one
			// and out of the prediction window, which means we don't know the speed
			// so we calculate REAL speed using future position we already know
			candidates[i].Speed = calculateShipSpeed(float64(speedCandidates[i+1].Time-candidates[i].Time), speedCandidates[i+1].Position, candidates[i].Position)
		}
	}

	// no matter where we start or end, rewind first ship
	candidates[0].Position = candidates[0].Position.Add(candidates[0].Speed.ScalarMultiply(float64(ps.Time - candidates[0].Time)))
	candidates[0].Time = ps.Time

	return candidates
}

func statusForDist(minDist float64) Status {
	if minDist < RedThreshold {
		return Red
	}
	if minDist < YellowThreshold {
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

	if relSpeedSq < epsilon {
		return rPos.Magnitude()
	}

	dotProduct := rPos.Dot(rVel)
	// Time of closest approach - painful math
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
