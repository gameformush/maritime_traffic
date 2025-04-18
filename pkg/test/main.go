package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const (
	// Simulation constants
	predictionTimeSeconds = 60.0
	epsilon               = 1e-9 // For floating point comparisons

	// Quadtree constants
	quadtreeNodeCapacity  = 4 // Max items in a leaf before splitting
	initialQuadtreeBounds = 1e9 // Root node goes from -bounds to +bounds
)

// --- Status Enum ---

type Status string

const (
	StatusGreen  Status = "green"
	StatusYellow Status = "yellow"
	StatusRed    Status = "red"
)

// --- Data Structures ---

type Position struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
}

// ShipState represents the current, live state of a ship
type ShipState struct {
	ID        string    `json:"id"` // Keep ID here for convenience in Quadtree points
	LastTime  int64     `json:"last_time"`
	X         int64     `json:"x"` // Current X
	Y         int64     `json:"y"` // Current Y
	Vx        float64   `json:"-"` // Velocity X (internal)
	Vy        float64   `json:"-"` // Velocity Y (internal)
	Speed     float64   `json:"last_speed"`
	Status    Status    `json:"last_status"`
	mu        sync.Mutex // Mutex to protect individual ship state updates if needed (finer-grained locking - not used with global lock)
	IsNew     bool       `json:"-"` // Flag to track if it's the very first report
}

// PositionRecord stores a snapshot for history
type PositionRecord struct {
	Time     int64    `json:"time"`
	Speed    float64  `json:"speed"`
	Position Position `json:"position"`
}

// --- Quadtree Implementation ---

type Rect struct {
	MinX, MinY, MaxX, MaxY float64
}

func (r Rect) ContainsPoint(x, y float64) bool {
	return x >= r.MinX && x < r.MaxX && y >= r.MinY && y < r.MaxY
}

func (r Rect) Intersects(other Rect) bool {
	return r.MinX < other.MaxX && r.MaxX > other.MinX &&
		r.MinY < other.MaxY && r.MaxY > other.MinY
}

// Point interface for Quadtree items
type Point interface {
	GetX() float64
	GetY() float64
	GetID() string // Needed to avoid self-collision check
}

// Implement Point interface for ShipState (using current coords)
func (s *ShipState) GetX() float64 { return float64(s.X) }
func (s *ShipState) GetY() float64 { return float64(s.Y) }
func (s *ShipState) GetID() string { return s.ID }

type QuadtreeNode struct {
	Bounds    Rect
	Points    []Point // Points within this node (if leaf or storing points internally)
	Children  [4]*QuadtreeNode // NW, NE, SW, SE
	IsLeaf    bool
	Capacity  int
	level     int // For debugging/depth limiting
}

func NewQuadtreeNode(bounds Rect, capacity, level int) *QuadtreeNode {
	return &QuadtreeNode{
		Bounds:   bounds,
		Points:   make([]Point, 0, capacity),
		IsLeaf:   true,
		Capacity: capacity,
		level:    level,
	}
}

func (n *QuadtreeNode) Subdivide() {
	if !n.IsLeaf {
		return // Already subdivided
	}
	n.IsLeaf = false
	cx := n.Bounds.MinX + (n.Bounds.MaxX-n.Bounds.MinX)/2
	cy := n.Bounds.MinY + (n.Bounds.MaxY-n.Bounds.MinY)/2
	nextLevel := n.level + 1

	// NW
	n.Children[0] = NewQuadtreeNode(Rect{n.Bounds.MinX, cy, cx, n.Bounds.MaxY}, n.Capacity, nextLevel)
	// NE
	n.Children[1] = NewQuadtreeNode(Rect{cx, cy, n.Bounds.MaxX, n.Bounds.MaxY}, n.Capacity, nextLevel)
	// SW
	n.Children[2] = NewQuadtreeNode(Rect{n.Bounds.MinX, n.Bounds.MinY, cx, cy}, n.Capacity, nextLevel)
	// SE
	n.Children[3] = NewQuadtreeNode(Rect{cx, n.Bounds.MinY, n.Bounds.MaxX, cy}, n.Capacity, nextLevel)

	// Redistribute points from this node to children
	oldPoints := n.Points
	n.Points = make([]Point, 0, n.Capacity) // Clear points from internal node
	for _, p := range oldPoints {
		n.insertInternal(p)
	}
}

func (n *QuadtreeNode) insertInternal(p Point) bool {
	if !n.Bounds.ContainsPoint(p.GetX(), p.GetY()) {
		// This should ideally be handled by dynamic expansion or checked before calling insert
		log.Printf("WARN: Point (%f, %f) outside node bounds %+v", p.GetX(), p.GetY(), n.Bounds)
		return false // Point outside this node's boundary
	}

	if n.IsLeaf {
		if len(n.Points) < n.Capacity || n.level > 20 { // Add depth limit to prevent infinite recursion with coincident points
			n.Points = append(n.Points, p)
			return true
		} else {
			n.Subdivide()
			// Fall through to insert into children
		}
	}

	// Insert into the correct child
	for _, child := range n.Children {
		if child != nil && child.Bounds.ContainsPoint(p.GetX(), p.GetY()) {
			return child.insertInternal(p)
		}
	}
    // Should not happen if point is within bounds and subdivision logic is correct
    log.Printf("ERROR: Point (%f, %f) could not be placed in any child of node %+v", p.GetX(), p.GetY(), n.Bounds)
	return false
}


func (n *QuadtreeNode) Remove(p Point, x, y float64) bool {
	if !n.Bounds.ContainsPoint(x, y) {
		return false // Point not within this node's boundary
	}

	if n.IsLeaf {
		newPoints := make([]Point, 0, len(n.Points))
		found := false
		for _, point := range n.Points {
			if point.GetID() == p.GetID() && math.Abs(point.GetX()-x) < epsilon && math.Abs(point.GetY()-y) < epsilon {
                 found = true
				 // Don't add it back
			} else {
				newPoints = append(newPoints, point)
			}
		}
		n.Points = newPoints
		return found
	}

	// Recurse into children
	for _, child := range n.Children {
		if child != nil && child.Bounds.ContainsPoint(x, y) {
			if child.Remove(p, x, y) {
                // Optional: Prune empty branches (add complexity)
				return true
			}
		}
	}
	return false
}


func (n *QuadtreeNode) QueryRegion(region Rect, results *[]Point) {
	if !n.Bounds.Intersects(region) {
		return
	}

	if n.IsLeaf {
		for _, p := range n.Points {
			if region.ContainsPoint(p.GetX(), p.GetY()) {
				*results = append(*results, p)
			}
		}
		return
	}

	// Recurse into children
	for _, child := range n.Children {
        if child != nil {
		    child.QueryRegion(region, results)
        }
	}
    // Also check points stored at internal nodes if design allows (ours doesn't currently)
}

// --- Global State ---

type GlobalState struct {
	shipStates  map[string]*ShipState // Map ship ID -> current state
	shipHistory map[string][]PositionRecord // Map ship ID -> historical records
	quadtree    *QuadtreeNode
	mu          sync.RWMutex // Protects all shared access
}

func NewGlobalState() *GlobalState {
	bounds := Rect{
		MinX: -initialQuadtreeBounds, MinY: -initialQuadtreeBounds,
		MaxX: initialQuadtreeBounds, MaxY: initialQuadtreeBounds,
	}
	return &GlobalState{
		shipStates:  make(map[string]*ShipState),
		shipHistory: make(map[string][]PositionRecord),
		quadtree:    NewQuadtreeNode(bounds, quadtreeNodeCapacity, 0),
	}
}

// --- Collision Logic ---

func calculateMinDistance(s1, s2 *ShipState, duration float64) float64 {
	// Relative position and velocity
	deltaX := s1.X - s2.X
	deltaY := s1.Y - s2.Y
	deltaVx := s1.Vx - s2.Vx
	deltaVy := s1.Vy - s2.Vy

	// Squared relative speed
	relSpeedSq := deltaVx*deltaVx + deltaVy*deltaVy

	// If relative speed is zero, distance is constant
	if relSpeedSq < epsilon {
		return math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY))
	}

	// Time of closest approach (t_min for d^2(t))
	// t = - (delta_x*delta_vx + delta_y*delta_vy) / (delta_vx^2 + delta_vy^2)
	dotProduct := float64(deltaX)*deltaVx + float64(deltaY)*deltaVy
	tMin := -dotProduct / relSpeedSq

	var minDistSq float64

	// Check distance at t=0, t=duration, and t=tMin if within [0, duration]
	distSqT0 := float64(deltaX*deltaX + deltaY*deltaY)

	// Position at time t: p(t) = p0 + v*t
	x1Tdur := float64(s1.X) + s1.Vx*duration
	y1Tdur := float64(s1.Y) + s1.Vy*duration
	x2Tdur := float64(s2.X) + s2.Vx*duration
	y2Tdur := float64(s2.Y) + s2.Vy*duration
	distSqTdur := math.Pow(x1Tdur-x2Tdur, 2) + math.Pow(y1Tdur-y2Tdur, 2)

	minDistSq = math.Min(distSqT0, distSqTdur)

	if tMin > 0 && tMin < duration {
		// Position at time tMin
		x1Tmin := float64(s1.X) + s1.Vx*tMin
		y1Tmin := float64(s1.Y) + s1.Vy*tMin
		x2Tmin := float64(s2.X) + s2.Vx*tMin
		y2Tmin := float64(s2.Y) + s2.Vy*tMin
		distSqTmin := math.Pow(x1Tmin-x2Tmin, 2) + math.Pow(y1Tmin-y2Tmin, 2)
		minDistSq = math.Min(minDistSq, distSqTmin)
	}

	// Handle potential negative due to float errors, although unlikely with sq distances
    if minDistSq < 0 { return 0 }
	return math.Sqrt(minDistSq)
}

// --- API Handlers ---

type PostPositionRequest struct {
	Time int64 `json:"time"`
	X    int64 `json:"x"`
	Y    int64 `json:"y"`
}

type PostPositionResponse struct {
	Time  int64  `json:"time"`
	X     int64  `json:"x"`
	Y     int64  `json:"y"`
	Speed float64 `json:"speed"`
	Status Status `json:"status"`
}

func (gs *GlobalState) handlePostPosition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shipID := vars["id"]
	if shipID == "" {
		http.Error(w, `{"error": "missing ship id"}`, http.StatusBadRequest)
		return
	}

	var req PostPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "invalid request body: %v"}`, err), http.StatusBadRequest)
		return
	}

	currentTime := time.Now().Unix()

	gs.mu.Lock() // ***** GLOBAL WRITE LOCK *****
	defer gs.mu.Unlock()

	ship, exists := gs.shipStates[shipID]
	isFirstReport := !exists

	// --- Timestamp Validation ---
	if req.Time > currentTime {
		http.Error(w, `{"error": "time is in the future"}`, http.StatusUnprocessableEntity)
		return
	}
	if exists && req.Time <= ship.LastTime {
		http.Error(w, `{"error": "time out of range or not monotonically increasing"}`, http.StatusUnprocessableEntity)
		return
	}

	var currentSpeed float64
	var currentStatus Status
	var currentVx, currentVy float64
    var oldX, oldY int64 // Needed for Quadtree remove

	if isFirstReport {
		// --- Initial Placement ---
		currentSpeed = 0.0
		currentVx = 0.0
		currentVy = 0.0
		currentStatus = StatusGreen

		// Check for collision at spawn point (d(0) == 0)
		queryBounds := Rect{ // Tiny box around the exact point
			MinX: float64(req.X) - epsilon, MinY: float64(req.Y) - epsilon,
			MaxX: float64(req.X) + epsilon, MaxY: float64(req.Y) + epsilon,
		}
		candidates := make([]Point, 0, 10)
		gs.quadtree.QueryRegion(queryBounds, &candidates)

		for _, p := range candidates {
			otherShip := p.(*ShipState) // Assuming Point is always *ShipState
			if otherShip.X == req.X && otherShip.Y == req.Y {
				currentStatus = StatusRed
				log.Printf("INFO: Ship %s spawned directly on ship %s at (%d, %d). Status RED.", shipID, otherShip.ID, req.X, req.Y)
				break // Red overrides everything
			}
		}

		ship = &ShipState{
			ID:       shipID,
			LastTime: req.Time,
			X:        req.X,
			Y:        req.Y,
			Vx:       currentVx,
			Vy:       currentVy,
			Speed:    currentSpeed,
			Status:   currentStatus,
            IsNew:    true,
		}
		gs.shipStates[shipID] = ship
		gs.shipHistory[shipID] = make([]PositionRecord, 0, 10) // Initialize history slice

	} else {
        // --- Subsequent Report ---
        oldX, oldY = ship.X, ship.Y // Store previous position for Quadtree removal

		// Calculate velocity and speed
		deltaTime := float64(req.Time - ship.LastTime)
		if deltaTime < epsilon { // Should be caught by validation, but safety check
            http.Error(w, `{"error": "time delta is zero or negative"}`, http.StatusUnprocessableEntity)
            return
		}
		deltaX := float64(req.X - ship.X)
		deltaY := float64(req.Y - ship.Y)

		currentVx = deltaX / deltaTime
		currentVy = deltaY / deltaTime
		currentSpeed = math.Sqrt(currentVx*currentVx + currentVy*currentVy)

		// Remove old position from Quadtree BEFORE updating ship state
		// Need to pass the *old* coordinates for removal
        tempShipRefForRemoval := &ShipState{ID: shipID, X: oldX, Y: oldY} // Temporary ref with old coords
		if !gs.quadtree.Remove(tempShipRefForRemoval, float64(oldX), float64(oldY)) {
             log.Printf("WARN: Failed to remove ship %s from Quadtree at previous position (%d, %d)", shipID, oldX, oldY)
             // Continue anyway, Quadtree might have stale point, but state will update
        }

		// Update ship state (status updated later)
		ship.LastTime = req.Time
		ship.X = req.X
		ship.Y = req.Y
		ship.Vx = currentVx
		ship.Vy = currentVy
		ship.Speed = currentSpeed
        ship.IsNew = false
	}

	// Add/Update history record
	historyRecord := PositionRecord{
		Time:  req.Time,
		Speed: currentSpeed,
		Position: Position{X: req.X, Y: req.Y},
	}
	gs.shipHistory[shipID] = append(gs.shipHistory[shipID], historyRecord)

	// Insert new/updated ship position into Quadtree
    // Use the *updated* ship state pointer
	if !gs.quadtree.insertInternal(ship) {
         // This could happen if ship is outside initial large bounds
         log.Printf("ERROR: Failed to insert ship %s into Quadtree at (%d, %d)", shipID, req.X, req.Y)
         // Decide how to handle: error response? proceed with potentially inaccurate collision check?
         http.Error(w, `{"error": "ship position outside system bounds"}`, http.StatusInternalServerError)
         return
    }


	// --- Collision Detection (Only if not red from spawning) ---
	finalStatus := ship.Status // Use initial status (Green or Red from spawn)
	if finalStatus != StatusRed {
        finalStatus = StatusGreen // Reset to Green if it wasn't Red from spawning

		// Determine query region based on 60s prediction + buffer
        // Simple buffer approach: add max travel distance
        maxTravel := 100.0 * predictionTimeSeconds // Max speed * time
        buffer := maxTravel // A large buffer

        // BBox of ship's path prediction
        minPredX := math.Min(float64(ship.X), float64(ship.X)+ship.Vx*predictionTimeSeconds)
        maxPredX := math.Max(float64(ship.X), float64(ship.X)+ship.Vx*predictionTimeSeconds)
        minPredY := math.Min(float64(ship.Y), float64(ship.Y)+ship.Vy*predictionTimeSeconds)
        maxPredY := math.Max(float64(ship.Y), float64(ship.Y)+ship.Vy*predictionTimeSeconds)

		queryBounds := Rect{
			MinX: minPredX - buffer,
			MinY: minPredY - buffer,
			MaxX: maxPredX + buffer,
			MaxY: maxPredY + buffer,
		}

		candidates := make([]Point, 0, 100) // Preallocate some space
		gs.quadtree.QueryRegion(queryBounds, &candidates)

		//log.Printf("DEBUG: Ship %s query bounds %+v found %d candidates", shipID, queryBounds, len(candidates))


		for _, p := range candidates {
            otherShip := p.(*ShipState) // Type assertion
			if otherShip.ID == shipID {
				continue // Don't check against self
			}

			minDist := calculateMinDistance(ship, otherShip, predictionTimeSeconds)

			// Check status thresholds (Red overrides Yellow)
			if math.Abs(minDist - 0.0) < epsilon {
				finalStatus = StatusRed
				log.Printf("INFO: Ship %s potential collision (d_min=0) with %s. Status RED.", shipID, otherShip.ID)
				break // Red found, no need to check further
			} else if math.Abs(minDist - 1.0) < epsilon {
                 // Only set Yellow if not already Red
                 if finalStatus != StatusRed {
				    finalStatus = StatusYellow
                    // Continue checking in case a Red condition exists with another ship
                    log.Printf("INFO: Ship %s potential close call (d_min=1) with %s. Status YELLOW.", shipID, otherShip.ID)
                 }
			}
		}
	}

	// Update final status in state
	ship.Status = finalStatus

	// Prepare and send response
	resp := PostPositionResponse{
		Time:  req.Time,
		X:     req.X,
		Y:     req.Y,
		Speed: currentSpeed,
		Status: finalStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// Response structure for GET /v1/api/ships
type GetAllShipsResponse struct {
	Ships []*ShipState `json:"ships"`
}

func (gs *GlobalState) handleGetAllShips(w http.ResponseWriter, r *http.Request) {
	gs.mu.RLock() // ***** GLOBAL READ LOCK *****
	defer gs.mu.RUnlock()

	// Note: Returning all ships might be huge! Consider pagination in real app.
	allShips := make([]*ShipState, 0, len(gs.shipStates))
	for _, ship := range gs.shipStates {
        // Create a copy to avoid race conditions if we remove the ship.mu later
        shipCopy := &ShipState{
            ID:       ship.ID,
            LastTime: ship.LastTime,
            // X/Y not needed in this response per API spec? Let's add LastPosition instead.
            Speed:    ship.Speed,
            Status:   ship.Status,
            // Add LastPosition field if needed by spec. For now, matching example.
        }
		allShips = append(allShips, shipCopy) // Append the copy
	}

    // Let's adjust the output to match the spec exactly
    type ShipInfo struct {
        ID          string   `json:"id"`
        LastTime    int64    `json:"last_time"`
        LastStatus  Status   `json:"last_status"`
        LastSpeed   float64  `json:"last_speed"`
        LastPosition Position `json:"last_position"`
    }
    shipInfos := make([]ShipInfo, 0, len(gs.shipStates))
    for _, ship := range gs.shipStates {
        shipInfos = append(shipInfos, ShipInfo{
            ID:       ship.ID,
            LastTime: ship.LastTime,
            LastStatus:  ship.Status,
            LastSpeed:   ship.Speed,
            LastPosition: Position{X: ship.X, Y: ship.Y},
        })
    }


	resp := map[string][]ShipInfo{"ships": shipInfos} // Match spec {"ships": [...]}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Response structure for GET /v1/api/ships/:id
type GetShipByIDResponse struct {
	ID        string           `json:"id"`
	Positions []PositionRecord `json:"positions"`
}

func (gs *GlobalState) handleGetShipByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shipID := vars["id"]

	gs.mu.RLock() // ***** GLOBAL READ LOCK *****
	defer gs.mu.RUnlock()

	history, exists := gs.shipHistory[shipID]
	if !exists {
		http.Error(w, `{"error": "ship not found"}`, http.StatusNotFound)
		return
	}

	// Return a copy of the history slice
	historyCopy := make([]PositionRecord, len(history))
	copy(historyCopy, history)

	resp := GetShipByIDResponse{
		ID:        shipID,
		Positions: historyCopy,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (gs *GlobalState) handleFlush(w http.ResponseWriter, r *http.Request) {
	gs.mu.Lock() // ***** GLOBAL WRITE LOCK *****
	defer gs.mu.Unlock()

	// Clear state
	gs.shipStates = make(map[string]*ShipState)
	gs.shipHistory = make(map[string][]PositionRecord)

	// Re-initialize Quadtree
	bounds := Rect{
		MinX: -initialQuadtreeBounds, MinY: -initialQuadtreeBounds,
		MaxX: initialQuadtreeBounds, MaxY: initialQuadtreeBounds,
	}
	gs.quadtree = NewQuadtreeNode(bounds, quadtreeNodeCapacity, 0)

	log.Println("INFO: All ship data flushed.")
	w.WriteHeader(http.StatusNoContent)
}


// --- Main Function ---

func main() {
	globalState := NewGlobalState()

	r := mux.NewRouter()

	// API v1 routes
	apiV1 := r.PathPrefix("/v1/api").Subrouter()
	apiV1.HandleFunc("/ships/{id}/position", globalState.handlePostPosition).Methods("POST")
	apiV1.HandleFunc("/ships", globalState.handleGetAllShips).Methods("GET")
	apiV1.HandleFunc("/ships/{id}", globalState.handleGetShipByID).Methods("GET")
	apiV1.HandleFunc("/flush", globalState.handleFlush).Methods("POST")

	// Add a simple health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request){
        w.WriteHeader(http.StatusOK)
        fmt.Fprintln(w, "OK")
    }).Methods("GET")


	port := 8080
	log.Printf("Starting ship coordination server on port %d...\n", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), r))
}