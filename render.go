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

// drawRect draws a filled rectangle
func drawRect(dst *ebiten.Image, x, y, width, height float64, clr color.Color) {
	for py := int(y); py < int(y+height); py++ {
		for px := int(x); px < int(x+width); px++ {
			if px >= 0 && px < dst.Bounds().Dx() && py >= 0 && py < dst.Bounds().Dy() {
				dst.Set(px, py, clr)
			}
		}
	}
}

// drawRectOutline draws an outlined rectangle
func drawRectOutline(dst *ebiten.Image, x, y, width, height float64, clr color.Color) {
	// Top and bottom edges
	for px := int(x); px <= int(x+width); px++ {
		if px >= 0 && px < dst.Bounds().Dx() {
			py := int(y)
			if py >= 0 && py < dst.Bounds().Dy() {
				dst.Set(px, py, clr)
			}
			py = int(y + height)
			if py >= 0 && py < dst.Bounds().Dy() {
				dst.Set(px, py, clr)
			}
		}
	}
	// Left and right edges
	for py := int(y); py <= int(y+height); py++ {
		if py >= 0 && py < dst.Bounds().Dy() {
			px := int(x)
			if px >= 0 && px < dst.Bounds().Dx() {
				dst.Set(px, py, clr)
			}
			px = int(x + width)
			if px >= 0 && px < dst.Bounds().Dx() {
				dst.Set(px, py, clr)
			}
		}
	}
}