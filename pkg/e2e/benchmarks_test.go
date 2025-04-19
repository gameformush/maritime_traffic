package e2e

import (
	"fmt"
	"maritime_traffic/pkg/handlers"
	"math/rand/v2"
	"testing"
)

func BenchmarkPosition(b *testing.B) {
	client := NewClient(addr, port)

	shipsAmount := []int{1, 10, 100, 1000, 10_000, 100_000, 1_000_000}

	for _, N := range shipsAmount {
		b.Run(fmt.Sprintf("ships=%v", N), func(b *testing.B) {
			client.Flush()
			idsPool := IdsPool(N)
			time := 0

			b.ReportAllocs()
			b.ReportMetric(float64(N), "ships")
			b.ResetTimer()

			for b.Loop() {
				client.PositionShip(idsPool[rand.IntN(N)], time, handlers.Position{X: rand.Int(), Y: rand.Int()})
				time++
			}
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
