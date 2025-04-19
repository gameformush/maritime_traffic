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

func TestEvaluateTrafficStatus(t *testing.T) {
	tests := []struct {
		name           string
		history        map[string][]ShipPosition
		positionShip   PositionShip
		speed          Vector
		expectedStatus Status
	}{
		{
			name:           "No other ships",
			history:        make(map[string][]ShipPosition),
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 2, Y: 4}},
			speed:          Vector{X: 1, Y: 1},
			expectedStatus: Green,
		},
		{
			name: "Green status - ships far apart",
			history: map[string][]ShipPosition{
				"ship2": {
					{Time: 90, Position: Vector{X: 10, Y: 10}, Speed: Vector{X: 0, Y: 0}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 2, Y: 2}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Green,
		},
		{
			name: "Yellow status - ships within yellow threshold",
			history: map[string][]ShipPosition{
				"ship2": {
					{Time: 100, Position: Vector{X: 1.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Yellow,
		},
		{
			name: "Red status - ships very close",
			history: map[string][]ShipPosition{
				"ship2": {
					{Time: 100, Position: Vector{X: 0.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Red,
		},
		{
			name: "Same ship ID - should ignore itself",
			history: map[string][]ShipPosition{
				"ship1": {
					{Time: 90, Position: Vector{X: 0, Y: 0}, Speed: Vector{X: 5, Y: 5}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 5, Y: 5}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Green,
		},
		{
			name: "Future collision based on trajectories",
			history: map[string][]ShipPosition{
				"ship2": {
					{Time: 100, Position: Vector{X: 10, Y: 0}, Speed: Vector{X: -1, Y: 0}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}},
			speed:          Vector{X: 1, Y: 0}, // Moving right
			expectedStatus: Red,
		},
		{
			name: "Multiple ships - one causes Red status",
			history: map[string][]ShipPosition{
				"ship2": {
					{Time: 100, Position: Vector{X: 1.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
				},
				"ship3": {
					{Time: 100, Position: Vector{X: 0.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Red,
		},
		{
			name: "Time window consideration",
			history: map[string][]ShipPosition{
				"ship2": {
					{Time: 90, Position: Vector{X: 5.5, Y: 0}, Speed: Vector{X: 1, Y: 0}},
					{Time: 110, Position: Vector{X: 25, Y: 0}, Speed: Vector{X: 1, Y: 0}},
					{Time: 140, Position: Vector{X: 55, Y: 0}, Speed: Vector{X: 1, Y: 0}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 5, Y: 0}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Green,
		},
		{
			name:           "Tower collision",
			history:        make(map[string][]ShipPosition),
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Red,
		},
		{
			name:           "Tower proximity - yellow warning",
			history:        make(map[string][]ShipPosition),
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 1, Y: 1}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Yellow,
		},
		{
			name: "Edge of time window",
			history: map[string][]ShipPosition{
				"ship2": {
					{Time: 100 + predictionTimeSeconds, Position: Vector{X: 0.5, Y: 0}, Speed: Vector{X: 0, Y: 0}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}},
			speed:          Vector{X: 0, Y: 0},
			expectedStatus: Red,
		},
		{
			name: "Converging paths",
			history: map[string][]ShipPosition{
				"ship2": {
					{Time: 100, Position: Vector{X: 10, Y: 10}, Speed: Vector{X: -1, Y: -1}},
				},
			},
			positionShip:   PositionShip{ID: "ship1", Time: 100, Point: Vector{X: 0, Y: 0}},
			speed:          Vector{X: 1, Y: 1},
			expectedStatus: Red,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traffic := NewTraffic()
			traffic.History = tt.history

			status := traffic.evaluateTrafficStatus(tt.positionShip, tt.speed)
			assert.Equal(t, tt.expectedStatus, status, "Unexpected traffic status")
		})
	}
}
