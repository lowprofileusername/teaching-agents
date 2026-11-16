package main

import (
	"math"
	"time"
)

// Native travel uses elapsed time, independent of HTTP or animation frame rate.
// Fractional pixels accumulate so slow movement remains smooth on high-DPI screens.
type teacherRoam struct {
	last    time.Time
	heading float64
	turnAt  time.Time
	rx, ry  float64
}

func (m *teacherRoam) step(now time.Time, x, y, loX, loY, hiX, hiY int, speed float64, active bool) (int, int, string) {
	dt := now.Sub(m.last).Seconds()
	if m.last.IsZero() {
		dt = 0
	}
	m.last = now
	if !active {
		m.rx = 0
		m.ry = 0
		return x, y, ""
	}
	dt = math.Max(0, math.Min(dt, .05))
	if m.turnAt.IsZero() {
		m.heading = 3.55
		m.turnAt = now.Add(5 * time.Second)
	}
	if now.After(m.turnAt) {
		m.heading += .68
		m.turnAt = now.Add(7 * time.Second)
	}
	vx, vy := math.Cos(m.heading), math.Sin(m.heading)
	m.rx += vx * speed * dt
	m.ry += vy * speed * dt
	dx, dy := int(m.rx), int(m.ry)
	m.rx -= float64(dx)
	m.ry -= float64(dy)
	x += dx
	y += dy
	if x < loX || x > hiX {
		x = max(loX, min(hiX, x))
		m.heading = math.Pi - m.heading
		m.rx = 0
	}
	if y < loY || y > hiY {
		y = max(loY, min(hiY, y))
		m.heading = -m.heading
		m.ry = 0
	}
	direction := "right"
	if math.Cos(m.heading) < 0 {
		direction = "left"
	}
	return x, y, direction
}
