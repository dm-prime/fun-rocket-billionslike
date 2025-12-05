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

// drawPredictiveTrail draws the predicted path as a trail of segments with fading opacity
func (g *Game) drawPredictiveTrail(screen *ebiten.Image, positions []vec2, player *Ship, shipScreenCenter vec2) {
	if len(positions) <= 1 {
		return
	}

	// Draw trail segments
	for i := 0; i < len(positions)-1; i++ {
		p1 := positions[i]
		p2 := positions[i+1]

		// Transform world coordinates to screen coordinates (relative to player)
		offset1 := vec2{p1.x - player.pos.x, p1.y - player.pos.y}
		rot1 := rotatePoint(offset1, -player.angle)
		screenX1 := shipScreenCenter.x + rot1.x
		screenY1 := shipScreenCenter.y + rot1.y

		offset2 := vec2{p2.x - player.pos.x, p2.y - player.pos.y}
		rot2 := rotatePoint(offset2, -player.angle)
		screenX2 := shipScreenCenter.x + rot2.x
		screenY2 := shipScreenCenter.y + rot2.y

		// Calculate opacity based on distance along trail (fade from full to transparent)
		// Earlier segments are more opaque, later segments fade out
		progress := float64(i) / float64(len(positions)-2)
		opacity := 1.0 - progress*0.8 // Fade from 1.0 to 0.2

		// Create faded color
		fadedColor := color.NRGBA{
			R: colorVelocityVector.R,
			G: colorVelocityVector.G,
			B: colorVelocityVector.B,
			A: uint8(float64(colorVelocityVector.A) * opacity),
		}

		// Draw trail segment
		ebitenutil.DrawLine(screen, screenX1, screenY1, screenX2, screenY2, fadedColor)
	}
}
