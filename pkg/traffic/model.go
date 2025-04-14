package traffic

type Status string

const (
	Green  Status = "green"
	Yellow Status = "yellow"
	Red    Status = "red"
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ShipPosition struct {
	Time     int64 `json:"time"`
	Speed    int   `json:"speed"`
	Position Point `json:"position"`
}

type Ship struct {
	ID           string `json:"id"`
	LastSeen     string `json:"last_time"`
	LastStatus   Status `json:"last_status"`
	LastSpeed    int    `json:"last_speed"`
	LastPosition Point  `json:"last_position"`
}

type PositionShip struct {
	ID   string
	Time int64
	X    int
	Y    int
}

type PositionResult struct {
	Speed  int
	Status Status
}
