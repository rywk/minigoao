package typ

import "image"

// Point
type P struct {
	X, Y int32
}

func (p *P) Point() image.Point {
	return image.Pt(int(p.X), int(p.Y))
}

func (p *P) Add(x, y int32) P {
	return P{
		X: p.X + x,
		Y: p.Y + y,
	}
}

func (p *P) Set(x, y int32) {
	p.X, p.Y = x, y
}

func (p *P) In(r Rect) bool {
	if p.X >= r.Min.X && p.Y >= r.Min.Y && p.X < r.Max.X && p.Y < r.Max.Y {
		return true
	}
	return false
}
func (p *P) Out(r Rect) bool {
	return !p.In(r)
}

type Rect struct {
	Min, Max P
}

func (r *Rect) R() image.Rectangle {
	return image.Rect(int(r.Min.X), int(r.Min.Y), int(r.Max.X), int(r.Max.Y))
}

func (r *Rect) Smaller(q int32) Rect {
	return Rect{
		Min: P{X: r.Min.X + q, Y: r.Min.Y + q},
		Max: P{X: r.Max.X - q, Y: r.Max.Y - q},
	}
}

func (r *Rect) OnPoint(p P) Rect {
	return Rect{
		Min: P{X: p.X + r.Min.X, Y: p.Y + r.Min.Y},
		Max: P{X: p.X + r.Max.X, Y: p.Y + r.Max.Y},
	}
}

func (r *Rect) Move(p P) Rect {
	return Rect{
		Min: P{X: p.X + r.Min.X, Y: p.Y + r.Min.Y},
		Max: P{X: p.X + r.Max.X, Y: p.Y + r.Max.Y},
	}
}
