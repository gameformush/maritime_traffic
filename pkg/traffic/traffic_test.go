package traffic

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"
)

func generateHistory(size int) []ShipPosition {
	history := make([]ShipPosition, size)
	baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	for i := 0; i < size; i++ {
		history[i] = ShipPosition{Time: int(baseTime + int64(i*10))} // e.g., 10-second intervals
	}
	return history
}

// Generic benchmark runner for different sizes and functions
func benchmarkRewindFunc(b *testing.B, size int, rewindFunc func([]ShipPosition, PositionShip) ShipPosition) {
	history := generateHistory(size)
	ps := PositionShip{Time: history[size/2].Time - predictionTimeSeconds/2} // A time within the prediction window of the middle element

	for b.Loop() {
		_ = rewindFunc(history, ps)
	}
}

// Benchmark for rewindShipBinary
func BenchmarkRewind(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000, 100_000, 1_000_000, 10_000_000, 100_000_000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("name=binary/size=%d", size), func(b *testing.B) {
			benchmarkRewindFunc(b, size, rewindShipBinarySearch)
		})
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("name=rewind/size=%d", size), func(b *testing.B) {
			benchmarkRewindFunc(b, size, rewindShip)
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
	for i := 0; i < N; i++ {
		ids[i] = RandomShipID()
	}
	return ids
}

func RandomShipID() string {
	return fmt.Sprintf("ship-%d", rand.IntN(1000000))
}
