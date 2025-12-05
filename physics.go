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

// NPC behavior constants
const (
	npcDesiredDist = 80.0 // standoff distance from player
)

// updateNPC uses the same physics as the player - can only turn and thrust forward
func (g *Game) updateNPC(npc *Ship, player *Ship, dt float64) {
	npc.thrustThisFrame = false
	npc.turningThisFrame = false
	npc.turnDirection = 0

	// === Calculate situation ===

	// Vector to player
	dx := player.pos.x - npc.pos.x
	dy := player.pos.y - npc.pos.y
	dist := math.Hypot(dx, dy)

	// Relative velocity (NPC velocity minus player velocity)
	relVelX := npc.vel.x - player.vel.x
	relVelY := npc.vel.y - player.vel.y
	relSpeed := math.Hypot(relVelX, relVelY)

	// Closing speed: positive = getting closer
	closingSpeed := 0.0
	if dist > 1 {
		closingSpeed = (dx*relVelX + dy*relVelY) / dist
	}

	// Calculate stopping distance at current closing speed
	stoppingDist := 0.0
	if closingSpeed > 0 {
		stoppingDist = (closingSpeed * closingSpeed) / (2 * thrustAccel)
	}

	// === Decide what to do ===

	distToDesired := dist - npcDesiredDist

	// Should we brake? Only if we're approaching AND would overshoot
	wantToBrake := closingSpeed > 15 && stoppingDist > distToDesired*0.7 && dist < npcDesiredDist*4

	// Should we thrust? Only if we're too far away
	wantToThrust := dist > npcDesiredDist && !wantToBrake

	// === Calculate target heading ===

	var targetAngle float64
	if wantToBrake {
		// Point opposite to our relative velocity (retrograde relative to player)
		if relSpeed > 5 {
			targetAngle = math.Atan2(-relVelX, relVelY)
		} else {
			// Low relative speed - just face the player
			targetAngle = math.Atan2(dx, -dy)
		}
	} else {
		// Point toward intercept - lead the target
		leadTime := clamp(dist/250.0, 0.1, 1.5)
		targetX := player.pos.x + player.vel.x*leadTime
		targetY := player.pos.y + player.vel.y*leadTime
		tdx := targetX - npc.pos.x
		tdy := targetY - npc.pos.y
		targetAngle = math.Atan2(tdx, -tdy)
	}

	angleDiff := normalizeAngle(targetAngle - npc.angle)

	// === TURN INPUT (like player pressing A/D) ===

	turnThreshold := 0.08 // radians - dead zone to prevent jitter
	if math.Abs(angleDiff) > turnThreshold {
		npc.turningThisFrame = true
		if angleDiff > 0 {
			// Turn right
			npc.angularVel += angularAccel * dt
			npc.turnDirection = 1
		} else {
			// Turn left
			npc.angularVel -= angularAccel * dt
			npc.turnDirection = -1
		}
	}

	// === AUTO ANGULAR DAMPING (same as player when not pressing A/D) ===

	if !npc.turningThisFrame && math.Abs(npc.angularVel) > 0.01 {
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

	// Clamp angular velocity
	if npc.angularVel > maxAngularSpeed {
		npc.angularVel = maxAngularSpeed
	}
	if npc.angularVel < -maxAngularSpeed {
		npc.angularVel = -maxAngularSpeed
	}

	// Update angle
	npc.angle += npc.angularVel * dt

	// === THRUST INPUT (like player pressing W) ===

	alignThreshold := math.Pi / 4 // 45 degrees - only thrust when reasonably aligned
	forwardX := math.Sin(npc.angle)
	forwardY := -math.Cos(npc.angle)

	if wantToBrake && math.Abs(angleDiff) < alignThreshold {
		// Aligned with retrograde - thrust to slow down
		npc.vel.x += forwardX * thrustAccel * dt
		npc.vel.y += forwardY * thrustAccel * dt
		npc.thrustThisFrame = true
	} else if wantToThrust && math.Abs(angleDiff) < alignThreshold {
		// Aligned with target - thrust to pursue
		npc.vel.x += forwardX * thrustAccel * dt
		npc.vel.y += forwardY * thrustAccel * dt
		npc.thrustThisFrame = true
	}

	// === UPDATE POSITION ===

	npc.pos.x += npc.vel.x * dt
	npc.pos.y += npc.vel.y * dt
}

// clamp limits a value to a range
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
