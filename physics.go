package main

import (
	"math"
)

// updatePhysics applies input-driven forces, rotation, and retrograde logic.
// This function works for ALL ships - player or NPC - using the unified ShipInput interface.
func (g *Game) updatePhysics(ship *Ship, input ShipInput, dt float64) {
	ship.thrustThisFrame = false
	ship.turningThisFrame = false
	ship.turnDirection = 0
	ship.dampingAngularSpeed = false

	// Apply angular acceleration based on input
	if input.TurnLeft {
		ship.angularVel -= angularAccel * dt
		ship.turningThisFrame = true
		ship.turnDirection = -1
	}
	if input.TurnRight {
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

	if input.ThrustForward {
		ship.vel.x += forwardX * thrustAccel * dt
		ship.vel.y += forwardY * thrustAccel * dt
		ship.thrustThisFrame = true
	}

	// Retrograde burn mode
	if input.RetrogradeBurn {
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
		// Retrograde burn not active - immediately cancel retrograde mode
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

// NPC behavior constants (shared with npc_ai.go)
const (
	npcDesiredDist   = 800.0  // standoff distance from player
	npcReacquireDist = 2000.0 // distance at which we consider player lost
	npcMaxSpeed      = 1000.0 // maximum speed NPCs should maintain
)

// updateNPC generates input for NPC and applies physics - NO LONGER NEEDED, kept for compatibility
// This function is deprecated - NPCs now generate inputs in main loop
func (g *Game) updateNPC(npc *Ship, player *Ship, dt float64) {
	// This is now handled in main.go Update loop
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
