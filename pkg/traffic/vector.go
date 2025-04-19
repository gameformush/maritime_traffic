package traffic

import "math"

type Vector struct {
	X float64
	Y float64
}

func (v Vector) Normalize() Vector {
	magnitude := v.Magnitude()
	if magnitude == 0 {
		return Vector{0, 0}
	}

	return Vector{
		X: v.X / magnitude,
		Y: v.Y / magnitude,
	}
}

func (v Vector) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vector) MagnitudeSquared() float64 {
	return v.X*v.X + v.Y*v.Y
}

func (v Vector) ScalarMultiply(scalar float64) Vector {
	return Vector{
		X: v.X * scalar,
		Y: v.Y * scalar,
	}
}

func (v Vector) Dot(other Vector) float64 {
	return v.X*other.X + v.Y*other.Y
}

func (v Vector) Subtract(other Vector) Vector {
	return Vector{
		X: v.X - other.X,
		Y: v.Y - other.Y,
	}
}

func (v Vector) Add(other Vector) Vector {
	return Vector{
		X: v.X + other.X,
		Y: v.Y + other.Y,
	}
}
