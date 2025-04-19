package e2e

import (
	"maritime_traffic/pkg/handlers"
	"maritime_traffic/pkg/traffic"
	"testing"

	"github.com/stretchr/testify/assert"
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
				{Speed: 1, Status: traffic.Green},
			},
		},
		{
			name: "two ships standing still - yellow",
			positions: []PositionRequest{
				{ID: "123", Time: 123, X: 1, Y: 1},
				{ID: "345", Time: 124, X: 2, Y: 2},
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Yellow},
			},
		},
		{
			name: "two ships standing still - red",
			positions: []PositionRequest{
				{ID: "123", Time: 123, X: 1, Y: 1},
				{ID: "345", Time: 124, X: 1, Y: 1},
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Red},
			},
		},
		{
			name: "two ships standing still - green",
			positions: []PositionRequest{
				{ID: "123", Time: 123, X: 1, Y: 1},
				{ID: "345", Time: 124, X: 3, Y: 3},
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
			},
		},
		{
			name: "two ships parallel movement - green",
			positions: []PositionRequest{
				// 0 0 1 2 3
				// 0 X X . .
				// 1 Y Y . .
				// 2 . . . .
				// 3 . . . .
				{ID: "123", Time: 123, X: 1, Y: 0},
				{ID: "345", Time: 123, X: 0, Y: 0},
				{ID: "123", Time: 124, X: 1, Y: 1},
				{ID: "345", Time: 124, X: 0, Y: 1},
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Yellow},
				{Speed: 1, Status: traffic.Yellow},
				{Speed: 1, Status: traffic.Yellow},
			},
		},
		{
			name: "collision course - red",
			positions: []PositionRequest{
				{ID: "123", Time: 100, X: 0, Y: 0},
				{ID: "345", Time: 100, X: 10, Y: 0},
				{ID: "123", Time: 101, X: 1, Y: 0}, // Moving right
				{ID: "345", Time: 101, X: 9, Y: 0}, // Moving left
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
				{Speed: 1, Status: traffic.Red},
				{Speed: 1, Status: traffic.Red},
			},
		},
		{
			name: "near miss - yellow",
			positions: []PositionRequest{
				{ID: "123", Time: 100, X: 0, Y: 0},
				{ID: "345", Time: 100, X: 10, Y: 2},
				{ID: "123", Time: 101, X: 2, Y: 0}, // Moving right
				{ID: "345", Time: 101, X: 8, Y: 2}, // Moving left but offset
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
				{Speed: 2, Status: traffic.Yellow},
				{Speed: 2, Status: traffic.Yellow},
			},
		},
		{
			name: "crossing paths at different times - green",
			positions: []PositionRequest{
				{ID: "123", Time: 100, X: 0, Y: 0},
				{ID: "345", Time: 100, X: 10, Y: 10},
				{ID: "345", Time: 105, X: 5, Y: 15}, // Still far away
				{ID: "123", Time: 105, X: 5, Y: 5},  // Will pass through (5,5)
				{ID: "345", Time: 110, X: 5, Y: 5},  // Will reach (5,5) later
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
				{Speed: 1, Status: traffic.Green},
				{Speed: 1, Status: traffic.Green},
				{Speed: 2, Status: traffic.Green},
			},
		},
		{
			name: "perpendicular movement - red",
			positions: []PositionRequest{
				{ID: "123", Time: 100, X: 0, Y: 0},
				{ID: "345", Time: 100, X: 3, Y: 3},
				{ID: "123", Time: 101, X: 1, Y: 0}, // Moving east
				{ID: "345", Time: 101, X: 3, Y: 2}, // Moving south
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
				{Speed: 1, Status: traffic.Green},
				{Speed: 1, Status: traffic.Red},
			},
		},
		{
			name: "high speed ships - red",
			positions: []PositionRequest{
				{ID: "123", Time: 100, X: 0, Y: 0},
				{ID: "345", Time: 100, X: 100, Y: 0},
				{ID: "123", Time: 101, X: 20, Y: 0}, // Fast moving right
				{ID: "345", Time: 101, X: 80, Y: 0}, // Fast moving left
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
				{Speed: 20, Status: traffic.Red},
				{Speed: 20, Status: traffic.Red},
			},
		},
		{
			name: "ships pass and get further - red then green",
			positions: []PositionRequest{
				{ID: "123", Time: 100, X: 0, Y: 0},
				{ID: "345", Time: 100, X: 4, Y: 0},
				{ID: "123", Time: 101, X: 1, Y: 0}, // Moving right
				{ID: "345", Time: 101, X: 3, Y: 0}, // Moving left
				{ID: "123", Time: 102, X: 4, Y: 0}, // Now passed each other
				{ID: "345", Time: 102, X: 0, Y: 0}, // Now passed each other
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
				{Speed: 1, Status: traffic.Red},
				{Speed: 1, Status: traffic.Red},
				{Speed: 3, Status: traffic.Green},
				{Speed: 3, Status: traffic.Green},
			},
		},
		{
			name: "three ships - collision risk",
			positions: []PositionRequest{
				{ID: "123", Time: 100, X: 0, Y: 0},
				{ID: "345", Time: 100, X: 5, Y: 0},
				{ID: "678", Time: 100, X: 2, Y: 2},
				{ID: "123", Time: 102, X: 2, Y: 0}, // Moving right
				{ID: "345", Time: 102, X: 3, Y: 0}, // Moving left
				{ID: "678", Time: 102, X: 2, Y: 0}, // Moving down
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Green},
				{Speed: 1, Status: traffic.Red},
				{Speed: 1, Status: traffic.Red},
				{Speed: 1, Status: traffic.Red},
			},
		},
		{
			name: "don't overwrite yellow with green",
			positions: []PositionRequest{
				{ID: "123", Time: 100, X: 0, Y: 0},
				{ID: "345", Time: 100, X: 1, Y: 1},
				{ID: "678", Time: 100, X: 4, Y: 4},
				{ID: "abc", Time: 100, X: 0, Y: 1},
			},
			expectedResults: []traffic.PositionResult{
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Yellow},
				{Speed: 0, Status: traffic.Green},
				{Speed: 0, Status: traffic.Yellow},
			},
		},
	}

	client := NewClient(addr, port)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			client.Flush()
			results := make([]traffic.PositionResult, len(tt.positions))
			for i, pos := range tt.positions {
				result, err := client.PositionShip(pos.ID, pos.Time, handlers.Position{X: pos.X, Y: pos.Y})
				require.NoError(t, err)
				results[i] = result
			}
			assert.Equal(t, tt.expectedResults, results)
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
		_, err := client.PositionShip("123", 123, handlers.Position{
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
		_, err := client.PositionShip("123", 123, handlers.Position{
			X: 1.0,
			Y: 1.0,
		})
		require.NoError(t, err)
	})
}
