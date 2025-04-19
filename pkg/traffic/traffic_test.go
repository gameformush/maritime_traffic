package traffic

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func generateHistory(size int) []ShipPosition {
	history := make([]ShipPosition, size)
	baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	for i := range size {
		history[i] = ShipPosition{Time: int(baseTime + int64(i*10))}
	}
	return history
}

func benchmarkRewindFunc(b *testing.B, size int, rewindFunc func([]ShipPosition, PositionShip) []ShipPosition) {
	history := generateHistory(size)
	ps := PositionShip{Time: history[size/2].Time - predictionTimeSeconds/2}

	for b.Loop() {
		_ = rewindFunc(history, ps)
	}
}

func BenchmarkRewind(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000, 100_000, 1_000_000, 10_000_000, 100_000_000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("name=binary/size=%d", size), func(b *testing.B) {
			benchmarkRewindFunc(b, size, rewindShipBinarySearch)
		})
	}
}

func BenchmarkPosition(b *testing.B) {
	t := NewTraffic()
	sizes := []int{10, 100, 1000, 10000, 100_000, 1_000_000, 10_000_000, 100_000_000}

	for _, N := range sizes {
		b.Run(fmt.Sprintf("name=seq/size=%d", N), func(b *testing.B) {
			t.Flush()
			idsPool := IdsPool(N)
			time := 0

			b.ReportAllocs()
			b.ReportMetric(float64(N), "ships")
			b.ResetTimer()

			b.ResetTimer()
			for b.Loop() {
				_, _ = t.PositionShip(PositionShip{
					ID:    idsPool[rand.Int64N(int64(N))],
					Time:  time,
					Point: Vector{X: float64(rand.Int()), Y: float64(rand.Int())},
				})
				time++
			}
		})

		b.Run(fmt.Sprintf("name=parallel/size=%d", N), func(b *testing.B) {
			t.Flush()
			idsPool := IdsPool(N)
			time := 0

			b.ReportAllocs()
			b.ReportMetric(float64(N), "ships")
			b.ResetTimer()

			b.SetParallelism(10)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = t.PositionShip(PositionShip{
						ID:    idsPool[rand.Int64N(int64(N))],
						Time:  time,
						Point: Vector{X: float64(rand.Int()), Y: float64(rand.Int())},
					})
					time++
				}
			})
		})
	}
}

func IdsPool(N int) []string {
	ids := make([]string, N)
	for i := range N {
		ids[i] = RandomShipID()
	}
	return ids
}

func RandomShipID() string {
	return fmt.Sprintf("ship-%d", rand.IntN(1000000))
}

func TestRewindShipCalculatesSpeed(t *testing.T) {
	history := []ShipPosition{
		{Time: 90, Position: Vector{X: 0, Y: 0}},
		{Time: 100, Position: Vector{X: 10, Y: 10}},
		{Time: 120, Position: Vector{X: 20, Y: 20}},
		{Time: 150, Position: Vector{X: 50, Y: 50}, Speed: Vector{2, 2}},
	}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, []ShipPosition{
		{Time: 100, Position: Vector{X: 10, Y: 10}, Speed: Vector{X: 0.5, Y: 0.5}},
		{Time: 120, Position: Vector{X: 20, Y: 20}, Speed: Vector{X: 1, Y: 1}},
		{Time: 150, Position: Vector{X: 50, Y: 50}, Speed: Vector{X: 2, Y: 2}},
	}, result)
}

func TestRewindShipUseLastForSpeed(t *testing.T) {
	history := []ShipPosition{
		{Time: 90, Position: Vector{X: 0, Y: 0}, Speed: Vector{0, 0}},
		{Time: 110, Position: Vector{X: 10, Y: 10}, Speed: Vector{1, 1}},
		{Time: 120, Position: Vector{X: 20, Y: 20}, Speed: Vector{1, 1}},
		{Time: 200, Position: Vector{X: 500, Y: 500}, Speed: Vector{2, 2}},
	}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, []ShipPosition{
		{Time: 100, Position: Vector{X: 5, Y: 5}, Speed: Vector{X: 0.5, Y: 0.5}}, // moved to start of the window
		{Time: 110, Position: Vector{X: 10, Y: 10}, Speed: Vector{X: 1, Y: 1}},   // speed adjusted
		{Time: 120, Position: Vector{X: 20, Y: 20}, Speed: Vector{X: 6, Y: 6}},   // speed adjusted using future position
	}, result)
}

func TestRewindShipStillShips(t *testing.T) {
	history := []ShipPosition{
		{Time: 123, Position: Vector{X: 1, Y: 1}},
	}
	ps := PositionShip{Time: 124}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, []ShipPosition{
		{Time: 124, Position: Vector{X: 1, Y: 1}, Speed: Vector{X: 0, Y: 0}},
	}, result)
}

func TestRewindShipPositionAdjustment(t *testing.T) {
	// Test that first and last positions are properly adjusted
	history := []ShipPosition{
		{Time: 90, Position: Vector{X: 0, Y: 0}, Speed: Vector{X: 1, Y: 1}},
		{Time: 110, Position: Vector{X: 20, Y: 20}, Speed: Vector{X: 1, Y: 1}},
		{Time: 140, Position: Vector{X: 50, Y: 50}, Speed: Vector{X: 1, Y: 1}},
	}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	// Check first position is rewound to ps.Time
	assert.Equal(t, 100, result[0].Time)
	assert.Equal(t, Vector{X: 10, Y: 10}, result[0].Position) // 0,0 + (1,1) * 10
	assert.Equal(t, 140, result[len(result)-1].Time)
	assert.Equal(t, Vector{X: 50, Y: 50}, result[len(result)-1].Position) // last stays the same
}

func TestRewindShipSinglePosition(t *testing.T) {
	// Test with a single position in history
	history := []ShipPosition{
		{Time: 90, Position: Vector{X: 0, Y: 0}, Speed: Vector{X: 1, Y: 1}},
	}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, 100, result[0].Time)
	assert.Equal(t, Vector{X: 10, Y: 10}, result[0].Position) // 0,0 + (1,1) * 10
}

func TestRewindShipExactBoundaries(t *testing.T) {
	// Test with positions exactly at boundaries
	history := []ShipPosition{
		{Time: 100, Position: Vector{X: 0, Y: 0}, Speed: Vector{X: 1, Y: 1}},
		{Time: 160, Position: Vector{X: 60, Y: 60}, Speed: Vector{X: 1, Y: 1}},
	}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, 2, len(result))
	assert.Equal(t, 100, result[0].Time)
	assert.Equal(t, Vector{X: 0, Y: 0}, result[0].Position) // Exact match, no adjustment

	assert.Equal(t, 160, result[1].Time)
	assert.Equal(t, Vector{X: 60, Y: 60}, result[1].Position) // Exact match, no adjustment
}

func TestRewindShipComplexScenario(t *testing.T) {
	// More complex scenario with varying speeds and times
	history := []ShipPosition{
		{Time: 80, Position: Vector{X: 0, Y: 0}, Speed: Vector{X: 2, Y: 1}},
		{Time: 95, Position: Vector{X: 30, Y: 15}, Speed: Vector{X: 2, Y: 1}},
		{Time: 120, Position: Vector{X: 80, Y: 40}, Speed: Vector{X: 2, Y: 1}},
		{Time: 140, Position: Vector{X: 120, Y: 60}, Speed: Vector{X: 2, Y: 1}},
		{Time: 170, Position: Vector{X: 180, Y: 90}, Speed: Vector{X: 2, Y: 1}},
	}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, 3, len(result))

	// First position should be at time 100 with adjusted position
	assert.Equal(t, 100, result[0].Time)
	// Position at time 95 is (30,15), with speed (2,1), so 5 seconds later it's (30,15) + (2,1)*5 = (40,20)
	expectedPos := Vector{X: 40, Y: 20}
	assert.InDeltaf(t, expectedPos.X, result[0].Position.X, 0.001, "X position doesn't match expected")
	assert.InDeltaf(t, expectedPos.Y, result[0].Position.Y, 0.001, "Y position doesn't match expected")

	// Last position doesn't change
	assert.Equal(t, 140, result[len(result)-1].Time)
}

func TestRewindShipEdgeCases(t *testing.T) {
	// Test edge case where we have data
	// after prediction time but need to go back one for starting point
	history := []ShipPosition{
		{Time: 95, Position: Vector{X: 0, Y: 0}, Speed: Vector{X: 1, Y: 1}},
		{Time: 180, Position: Vector{X: 85, Y: 85}, Speed: Vector{X: 1, Y: 1}},
	}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, 1, len(result))
	assert.Equal(t, 100, result[0].Time)
	assert.Equal(t, Vector{X: 5, Y: 5}, result[0].Position) // 0,0 + (1,1) * 5
}

func TestRewindShipOutOfRange(t *testing.T) {
	history := []ShipPosition{
		{Time: 200, Position: Vector{X: 0, Y: 0}, Speed: Vector{X: 1, Y: 1}},
		{Time: 250, Position: Vector{X: 10, Y: 10}, Speed: Vector{X: 1, Y: 1}},
	}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, 0, len(result)) // No positions should be returned
}

func TestRewindShipEmptyHistory(t *testing.T) {
	history := []ShipPosition{}
	ps := PositionShip{Time: 100}

	result := rewindShipBinarySearch(history, ps)

	assert.Equal(t, 0, len(result)) // No positions should be returned
}

// EvaluateTrafficStatus tests

func TestEvaluateTrafficStatusNoOtherShips(t *testing.T) {
	traffic := NewTraffic()
	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 1, Y: 1}

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Green, status, "Status should be Green when there are no other ships")
}

func TestEvaluateTrafficStatusGreenStatus(t *testing.T) {
	traffic := NewTraffic()

	// Add another ship far away
	otherShipID := "ship2"
	traffic.History[otherShipID] = []ShipPosition{
		{Time: 90, Position: Vector{X: 10, Y: 10}, Speed: Vector{X: 0, Y: 0}},
	}

	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 0, Y: 0}

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Green, status, "Status should be Green when ships are far apart")
}

func TestEvaluateTrafficStatusYellowStatus(t *testing.T) {
	traffic := NewTraffic()

	// Position another ship close enough for Yellow but not Red
	otherShipID := "ship2"
	traffic.History[otherShipID] = []ShipPosition{
		{Time: 100, Position: Vector{X: 1.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
	}

	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 0, Y: 0}

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Yellow, status, "Status should be Yellow when ships are within YellowThreshold")
}

func TestEvaluateTrafficStatusRedStatus(t *testing.T) {
	traffic := NewTraffic()

	// Position another ship very close
	otherShipID := "ship2"
	traffic.History[otherShipID] = []ShipPosition{
		{Time: 100, Position: Vector{X: 0.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
	}

	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 0, Y: 0}

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Red, status, "Status should be Red when ships are within RedThreshold")
}

func TestEvaluateTrafficStatusSameShipID(t *testing.T) {
	traffic := NewTraffic()

	shipID := "ship1"
	traffic.History[shipID] = []ShipPosition{
		{Time: 90, Position: Vector{X: 0, Y: 0}, Speed: Vector{X: 0, Y: 0}},
	}

	ps := PositionShip{ID: shipID, Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 0, Y: 0}

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Green, status, "Status should be Green when only considering itself")
}

func TestEvaluateTrafficStatusFutureCollision(t *testing.T) {
	traffic := NewTraffic()

	// Ships will collide in the future based on their trajectories
	otherShipID := "ship2"
	traffic.History[otherShipID] = []ShipPosition{
		{Time: 100, Position: Vector{X: 10, Y: 0}, Speed: Vector{X: -1, Y: 0}}, // Moving left
	}

	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 1, Y: 0} // Moving right
	// They will meet at x=5 after 5 seconds

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Red, status, "Status should be Red when ships will collide in the future")
}

func TestEvaluateTrafficStatusMultipleShips(t *testing.T) {
	traffic := NewTraffic()

	// Add a ship that would cause Yellow status
	traffic.History["ship2"] = []ShipPosition{
		{Time: 100, Position: Vector{X: 1.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
	}

	// Add a ship that would cause Red status
	traffic.History["ship3"] = []ShipPosition{
		{Time: 100, Position: Vector{X: 0.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
	}

	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 0, Y: 0}

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Red, status, "Status should be Red when at least one ship causes Red status")
}

func TestEvaluateTrafficStatusTimeWindow(t *testing.T) {
	traffic := NewTraffic()

	// Add a ship with multiple positions in history
	otherShipID := "ship2"
	traffic.History[otherShipID] = []ShipPosition{
		{Time: 90, Position: Vector{X: 0.5, Y: 0}, Speed: Vector{X: 1, Y: 0}},
		{Time: 110, Position: Vector{X: 20, Y: 0}, Speed: Vector{X: 1, Y: 0}},
		{Time: 140, Position: Vector{X: 50, Y: 0}, Speed: Vector{X: 1, Y: 0}},
	}

	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 0, Y: 0}

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Green, status, "Status should consider ship positions at the evaluation time")
}

func TestEvaluateTrafficStatusEdgeOfTimeWindow(t *testing.T) {
	traffic := NewTraffic()

	// Add a ship at the edge of the prediction window
	traffic.History["ship2"] = []ShipPosition{
		{Time: 100 + int(predictionTimeSeconds), Position: Vector{X: 0.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
	}

	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 0, Y: 0}

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Red, status, "Status should be Red when ship is at the edge of time window and within RedThreshold")
}

func TestEvaluateTrafficStatusConvergingPaths(t *testing.T) {
	traffic := NewTraffic()

	// Ships initially far apart but converging
	traffic.History["ship2"] = []ShipPosition{
		{Time: 100, Position: Vector{X: 10, Y: 10}, Speed: Vector{X: -1, Y: -1}},
	}

	ps := PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}}
	speed := Vector{X: 1, Y: 1}

	// Ships will meet somewhere in the middle

	status := traffic.evaluateTrafficStatus(ps, speed)

	assert.Equal(t, Red, status, "Status should be Red when ships are on converging paths")
}
