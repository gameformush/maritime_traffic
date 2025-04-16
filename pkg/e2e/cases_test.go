package e2e

import (
	"maritime_traffic/pkg/traffic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOk(t *testing.T) {
	client := NewClient("http://localhost", port)

	t.Run("flush", func(t *testing.T) {
		err := client.Flush()
		require.NoError(t, err)
	})

	t.Run("get ships", func(t *testing.T) {
		_, err := client.GetShips()
		require.NoError(t, err)
	})

	t.Run("get ship", func(t *testing.T) {
		_, err := client.PositionShip("123", 123, traffic.Point{
			X: 1.0,
			Y: 1.0,
		})
		require.NoError(t, err)
		_, err = client.GetShip("123")
		require.NoError(t, err)
	})

	t.Run("position ship", func(t *testing.T) {
		_, err := client.PositionShip("123", 123, traffic.Point{
			X: 1.0,
			Y: 1.0,
		})
		require.NoError(t, err)
	})
}
