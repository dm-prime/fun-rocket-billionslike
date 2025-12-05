package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// updatePhysics applies input-driven forces, rotation, and retrograde logic.
func (g *Game) updatePhysics(ship *Ship, dt float64) {
	ship.thrustThisFrame = false
	ship.turningThisFrame = false
	ship.turnDirection = 0
	ship.dampingAngularSpeed = false

	// Apply angular acceleration based on input
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		ship.angularVel -= angularAccel * dt
		ship.turningThisFrame = true
		ship.turnDirection = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		ship.angularVel += angularAccel * dt
		ship.turningThisFrame = true
		ship.turnDirection = 1
	}

	// Clamp angular velocity to max speed
	if ship.angularVel > maxAngularSpeed {
		ship.angularVel = maxAngularSpeed
	}
	if ship.angularVel < -maxAngularSpeed {
		ship.angularVel = -maxAngularSpeed
	}

	// Automatically apply angular damping when no turn input (A/D not pressed)
	if !ship.turningThisFrame && math.Abs(ship.angularVel) > 0.01 {
		// Gradually reduce angular velocity
		if ship.angularVel > 0 {
			ship.angularVel -= angularDampingAccel * dt * 0.5
			if ship.angularVel < 0 {
				ship.angularVel = 0
			}
		} else {
			ship.angularVel += angularDampingAccel * dt * 0.5
			if ship.angularVel > 0 {
				ship.angularVel = 0
			}
		}
	}

	// Update ship angle based on angular velocity
	ship.angle += ship.angularVel * dt

	forwardX := math.Sin(ship.angle)
	forwardY := -math.Cos(ship.angle)

	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		ship.vel.x += forwardX * thrustAccel * dt
		ship.vel.y += forwardY * thrustAccel * dt
		ship.thrustThisFrame = true
	}

	// S key activates retrograde burn mode
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		if !ship.retrogradeMode {
			// Entering retrograde mode - calculate the fastest turn direction
			ship.retrogradeMode = true
			ship.retrogradeTurnDir = g.calculateFastestRetrogradeTurn(ship)
		}
		// Execute retrograde burn maneuver
		if ship.retrogradeMode {
			g.executeRetrogradeBurn(ship, dt)
		}
	} else {
		// S key not held - immediately cancel retrograde mode
		if ship.retrogradeMode {
			ship.retrogradeMode = false
			ship.retrogradeTurnDir = 0
		}
	}

	ship.pos.x += ship.vel.x * dt
	ship.pos.y += ship.vel.y * dt
}

// normalizeAngle normalizes an angle to the range [-π, π].
func normalizeAngle(angle float64) float64 {
	for angle > math.Pi {
		angle -= 2 * math.Pi
	}
	for angle < -math.Pi {
		angle += 2 * math.Pi
	}
	return angle
}

// estimateTurnTime estimates time to turn a given distance considering current angular velocity.
func estimateTurnTime(targetDist, currentAngVel, accel float64) float64 {
	if targetDist < 0 {
		targetDist = -targetDist
		currentAngVel = -currentAngVel
	}

	if currentAngVel >= 0 {
		if currentAngVel > 0 {
			stopDist := 0.5 * currentAngVel * currentAngVel / accel
			if stopDist >= targetDist {
				overshoot := stopDist - targetDist
				brakeTime := currentAngVel / accel
				returnTime := math.Sqrt(2*overshoot/accel) * 2
				return brakeTime + returnTime
			}
			remainingDist := targetDist - stopDist
			return currentAngVel/accel + math.Sqrt(4*remainingDist/accel)
		}
		return math.Sqrt(4 * targetDist / accel)
	}
	stopTime := -currentAngVel / accel
	stopDist := 0.5 * currentAngVel * currentAngVel / accel
	newTargetDist := targetDist + stopDist
	return stopTime + math.Sqrt(4*newTargetDist/accel)
}

// calculateFastestRetrogradeTurn determines which direction to turn for fastest retrograde alignment.
func (g *Game) calculateFastestRetrogradeTurn(ship *Ship) float64 {
	speed := math.Hypot(ship.vel.x, ship.vel.y)
	if speed < 5.0 {
		return 0
	}

	// Calculate retrograde angle (opposite to velocity)
	// Ship forward is (sin(angle), -cos(angle))
	targetAngle := math.Atan2(-ship.vel.x, ship.vel.y)
	angleDiff := normalizeAngle(targetAngle - ship.angle)

	// Calculate time for short path vs long path
	shortTime := estimateTurnTime(angleDiff, ship.angularVel, angularAccel)

	var longDist float64
	if angleDiff > 0 {
		longDist = angleDiff - 2*math.Pi
	} else {
		longDist = angleDiff + 2*math.Pi
	}
	longTime := estimateTurnTime(longDist, ship.angularVel, angularAccel)

	// Choose the faster direction
	if shortTime <= longTime {
		if angleDiff >= 0 {
			return 1.0 // turn right
		}
		return -1.0 // turn left
	}
	if longDist >= 0 {
		return 1.0
	}
	return -1.0
}

// executeRetrogradeBurn handles the retrograde burn maneuver each frame.
func (g *Game) executeRetrogradeBurn(ship *Ship, dt float64) {
	speed := math.Hypot(ship.vel.x, ship.vel.y)

	// Check if velocity is killed
	if speed < 2.0 {
		ship.retrogradeMode = false
		ship.retrogradeTurnDir = 0
		return
	}

	// Always recalculate target angle each frame based on current velocity
	targetAngle := math.Atan2(-ship.vel.x, ship.vel.y)
	angleDiff := normalizeAngle(targetAngle - ship.angle)

	// Continuously align against speed - always turn towards retrograde direction
	ship.turningThisFrame = true
	ship.dampingAngularSpeed = true

	// Determine turn direction based on angle difference
	if math.Abs(angleDiff) > 0.01 { // Small threshold to avoid jitter
		// Determine which direction to turn (shortest path)
		if angleDiff > 0 {
			ship.retrogradeTurnDir = 1.0 // turn right
		} else {
			ship.retrogradeTurnDir = -1.0 // turn left
		}
		ship.turnDirection = ship.retrogradeTurnDir

		// Apply angular acceleration to turn towards retrograde
		if ship.retrogradeTurnDir > 0 {
			ship.angularVel += angularAccel * dt
		} else {
			ship.angularVel -= angularAccel * dt
		}
	} else {
		// Very close to alignment - dampen angular velocity to maintain alignment
		if math.Abs(ship.angularVel) > 0.01 {
			if ship.angularVel > 0 {
				ship.angularVel -= angularDampingAccel * dt
				ship.turnDirection = -1
			} else {
				ship.angularVel += angularDampingAccel * dt
				ship.turnDirection = 1
			}
		}
	}

	// Always apply side thruster acceleration directly against current velocity
	// regardless of ship orientation
	if speed > 0.01 { // Avoid division by zero
		// Normalize velocity vector to get direction
		velDirX := ship.vel.x / speed
		velDirY := ship.vel.y / speed
		// Apply acceleration opposite to velocity (retrograde direction)
		ship.vel.x -= velDirX * sideThrustAccel * dt
		ship.vel.y -= velDirY * sideThrustAccel * dt
		ship.thrustThisFrame = true
	}
}

// updateNPC updates an NPC ship to follow the player
func (g *Game) updateNPC(npc *Ship, player *Ship, dt float64) {
	npc.thrustThisFrame = false
	npc.turningThisFrame = false
	npc.turnDirection = 0

	// Calculate direction to player
	dx := player.pos.x - npc.pos.x
	dy := player.pos.y - npc.pos.y
	dist := math.Hypot(dx, dy)

	// If very close, don't move
	if dist < 10 {
		// Apply damping to slow down
		npc.vel.x *= 0.95
		npc.vel.y *= 0.95
		npc.pos.x += npc.vel.x * dt
		npc.pos.y += npc.vel.y * dt
		return
	}

	// Calculate target angle (direction to player)
	targetAngle := math.Atan2(dx, -dy) // Using -dy because ship forward is (sin(angle), -cos(angle))
	angleDiff := normalizeAngle(targetAngle - npc.angle)

	// Turn towards the player
	if math.Abs(angleDiff) > 0.05 { // Small threshold to avoid jitter
		npc.turningThisFrame = true
		if angleDiff > 0 {
			npc.angularVel += angularAccel * dt
			npc.turnDirection = 1
		} else {
			npc.angularVel -= angularAccel * dt
			npc.turnDirection = -1
		}
	} else {
		// Close enough - dampen angular velocity
		if math.Abs(npc.angularVel) > 0.01 {
			if npc.angularVel > 0 {
				npc.angularVel -= angularDampingAccel * dt * 0.5
				if npc.angularVel < 0 {
					npc.angularVel = 0
				}
			} else {
				npc.angularVel += angularDampingAccel * dt * 0.5
				if npc.angularVel > 0 {
					npc.angularVel = 0
				}
			}
		}
	}

	// Clamp angular velocity to max speed
	if npc.angularVel > maxAngularSpeed {
		npc.angularVel = maxAngularSpeed
	}
	if npc.angularVel < -maxAngularSpeed {
		npc.angularVel = -maxAngularSpeed
	}

	// Update ship angle based on angular velocity
	npc.angle += npc.angularVel * dt

	// Accelerate towards player (thrust forward)
	forwardX := math.Sin(npc.angle)
	forwardY := -math.Cos(npc.angle)

	// Only thrust if reasonably aligned with target (within 45 degrees)
	if math.Abs(angleDiff) < math.Pi/4 {
		npc.vel.x += forwardX * thrustAccel * dt
		npc.vel.y += forwardY * thrustAccel * dt
		npc.thrustThisFrame = true
	}

	// Update position
	npc.pos.x += npc.vel.x * dt
	npc.pos.y += npc.vel.y * dt
}
