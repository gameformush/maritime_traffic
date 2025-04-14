package traffic

type Traffic struct {
}

func NewTraffic() *Traffic {
	return &Traffic{}
}

func (t *Traffic) GetShips() ([]Ship, error) {
	return []Ship{
		{ID: "1", LastSeen: "2023-10-01T12:00:00Z", LastStatus: Green, LastSpeed: 20, LastPosition: Point{X: 100, Y: 200}},
		{ID: "2", LastSeen: "2023-10-01T12:05:00Z", LastStatus: Yellow, LastSpeed: 15, LastPosition: Point{X: 150, Y: 250}},
	}, nil
}

func (t *Traffic) GetShipPositions(id string) ([]ShipPosition, error) {
	return []ShipPosition{
		{
			Time:  123,
			Speed: 12,
			Position: Point{
				X: 1,
				Y: 2,
			},
		},
	}, nil
}

func (t *Traffic) PositionShip(ps PositionShip) (PositionResult, error) {
	return PositionResult{
		Speed:  10,
		Status: Green,
	}, nil
}
