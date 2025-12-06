package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// predictFuturePath simulates the ship's movement forward in time based on current state and inputs
func (g *Game) predictFuturePath(ship *Ship, input ShipInput) []vec2 {
	// Create a copy of ship state for simulation
	simPos := vec2{ship.pos.x, ship.pos.y}
	simVel := vec2{ship.vel.x, ship.vel.y}
	simAngle := ship.angle
	simAngularVel := ship.angularVel

	// Calculate time step
	dt := predictiveTrailUpdateRate
	steps := predictiveTrailSegmentCount

	// Store predicted positions
	positions := make([]vec2, 0, steps+1)
	positions = append(positions, simPos) // Start at current position

	// Simulate forward in time
	for i := 0; i < steps; i++ {
		// Apply angular acceleration based on input
		if input.TurnLeft {
			simAngularVel -= angularAccel * dt
		}
		if input.TurnRight {
			simAngularVel += angularAccel * dt
		}

		// Clamp angular velocity to max speed
		if simAngularVel > maxAngularSpeed {
			simAngularVel = maxAngularSpeed
		}
		if simAngularVel < -maxAngularSpeed {
			simAngularVel = -maxAngularSpeed
		}

		// Automatically apply angular damping when no turn input
		if !input.TurnLeft && !input.TurnRight && math.Abs(simAngularVel) > 0.01 {
			if simAngularVel > 0 {
				simAngularVel -= angularDampingAccel * dt * 0.5
				if simAngularVel < 0 {
					simAngularVel = 0
				}
			} else {
				simAngularVel += angularDampingAccel * dt * 0.5
				if simAngularVel > 0 {
					simAngularVel = 0
				}
			}
		}

		// Update ship angle based on angular velocity
		simAngle += simAngularVel * dt

		forwardX := math.Sin(simAngle)
		forwardY := -math.Cos(simAngle)

		// Apply thrust
		if input.ThrustForward {
			simVel.x += forwardX * thrustAccel * dt
			simVel.y += forwardY * thrustAccel * dt
		}

		// Handle retrograde burn
		if input.RetrogradeBurn {
			speed := math.Hypot(simVel.x, simVel.y)
			if speed > 0.01 {
				// Apply retrograde burn (side thruster against velocity)
				velDirX := simVel.x / speed
				velDirY := simVel.y / speed
				simVel.x -= velDirX * sideThrustAccel * dt
				simVel.y -= velDirY * sideThrustAccel * dt
			}
		}

		// Update position
		simPos.x += simVel.x * dt
		simPos.y += simVel.y * dt

		positions = append(positions, vec2{simPos.x, simPos.y})
	}

	return positions
}

// drawPredictiveTrailInRadar draws the predicted path in radar space as a trail of segments with fading opacity
func (g *Game) drawPredictiveTrailInRadar(screen *ebiten.Image, positions []vec2, ship *Ship, player *Ship, center vec2, scale float64, radarRadius float64, trailColor color.NRGBA) {
	if len(positions) <= 1 {
		return
	}

	edgeLimit := radarRadius - radarEdgeMargin

	// Draw trail segments
	for i := 0; i < len(positions)-1; i++ {
		p1 := positions[i]
		p2 := positions[i+1]

		// Transform world coordinates to radar coordinates (relative to player position)
		dx1 := p1.x - player.pos.x
		dy1 := p1.y - player.pos.y
		rotated1 := rotatePoint(vec2{dx1, dy1}, -player.angle)
		rx1 := rotated1.x * scale
		ry1 := rotated1.y * scale

		dx2 := p2.x - player.pos.x
		dy2 := p2.y - player.pos.y
		rotated2 := rotatePoint(vec2{dx2, dy2}, -player.angle)
		rx2 := rotated2.x * scale
		ry2 := rotated2.y * scale

		// Clamp to radar edge if needed
		if edgeDist1 := math.Hypot(rx1, ry1); edgeDist1 > edgeLimit {
			f := edgeLimit / edgeDist1
			rx1 *= f
			ry1 *= f
		}
		if edgeDist2 := math.Hypot(rx2, ry2); edgeDist2 > edgeLimit {
			f := edgeLimit / edgeDist2
			rx2 *= f
			ry2 *= f
		}

		// Calculate opacity based on distance along trail (fade from full to transparent)
		// Earlier segments are more opaque, later segments fade out
		progress := float64(i) / float64(len(positions)-2)
		opacity := 1.0 - progress*0.8 // Fade from 1.0 to 0.2

		// Create faded color
		fadedColor := color.NRGBA{
			R: trailColor.R,
			G: trailColor.G,
			B: trailColor.B,
			A: uint8(float64(trailColor.A) * opacity),
		}

		// Draw trail segment in radar space
		ebitenutil.DrawLine(
			screen,
			center.x+rx1,
			center.y+ry1,
			center.x+rx2,
			center.y+ry2,
			fadedColor,
		)
	}
}
