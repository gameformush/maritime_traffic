package e2e

import (
	"maritime_traffic/pkg/traffic"
	"testing"

	"github.com/stretchr/testify/require"
)

type PositionRequest struct {
	ID   string
	Time int
	X    int
	Y    int
}

func TestBasicCases(t *testing.T) {
	tests := []struct {
		name            string
		positions       []PositionRequest
		expectedResults []traffic.PositionResult
	}{
		{
			name:            "happy path",
			positions:       []PositionRequest{{ID: "123", Time: 123, X: 1, Y: 1}},
			expectedResults: []traffic.PositionResult{{Speed: 0, Status: traffic.Green}},
		},
		{
			name: "two positions",
			positions: []PositionRequest{
				{ID: "123", Time: 123, X: 1, Y: 1},
				{ID: "123", Time: 124, X: 2, Y: 2},
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 2, Status: traffic.Green},
			},
		},
	}

	client := NewClient(addr, port)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			client.Flush()
			for i, pos := range tt.positions {
				result, err := client.PositionShip(pos.ID, pos.Time, traffic.Point{X: pos.X, Y: pos.Y})
				require.NoError(t, err)
				require.Equal(t, tt.expectedResults[i], result)
			}
		})
	}
}

func TestOk(t *testing.T) {
	client := NewClient(addr, port)

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

	t.Run("get ship not found", func(t *testing.T) {
		client.Flush()
		_, err := client.GetShip("123")
		require.Error(t, err)
	})

	t.Run("position ship", func(t *testing.T) {
		_, err := client.PositionShip("123", 123, traffic.Point{
			X: 1.0,
			Y: 1.0,
		})
		require.NoError(t, err)
	})
}
