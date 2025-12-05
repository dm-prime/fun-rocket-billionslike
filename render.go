package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// rotatePoint rotates a point around the origin by the given angle (in radians)
func rotatePoint(p vec2, angle float64) vec2 {
	sinA := math.Sin(angle)
	cosA := math.Cos(angle)
	return vec2{
		x: p.x*cosA - p.y*sinA,
		y: p.x*sinA + p.y*cosA,
	}
}

// drawCircle draws a simple filled circle using points around the circumference
func drawCircle(dst *ebiten.Image, cx, cy, radius float64, clr color.Color) {
	// Very cheap filled circle for the simple dust field.
	steps := int(radius*4 + 4)
	for i := 0; i < steps; i++ {
		angle := float64(i) / float64(steps) * 2 * math.Pi
		x := cx + math.Cos(angle)*radius
		y := cy + math.Sin(angle)*radius
		dst.Set(int(x), int(y), clr)
	}
}
