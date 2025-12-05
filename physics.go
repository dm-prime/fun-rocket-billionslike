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
	npcDesiredDist      = 100.0       // standoff distance from player
	npcInterceptLead    = 1.5         // seconds to lead target prediction
	npcThrustFactor     = 0.85        // NPC thrust multiplier (can almost match player)
	npcAlignThreshold   = 0.03        // radians, angle considered "aligned"
	npcBrakeAlignThresh = math.Pi / 6 // must be within this to brake effectively
)

// updateNPC updates an NPC ship with intelligent pursuit/intercept behavior
func (g *Game) updateNPC(npc *Ship, player *Ship, dt float64) {
	npc.thrustThisFrame = false
	npc.turningThisFrame = false
	npc.turnDirection = 0

	// === PHASE 1: Calculate pursuit geometry ===

	// Relative position
	dx := player.pos.x - npc.pos.x
	dy := player.pos.y - npc.pos.y
	dist := math.Hypot(dx, dy)

	// Relative velocity (positive = NPC moving toward player)
	relVelX := npc.vel.x - player.vel.x
	relVelY := npc.vel.y - player.vel.y
	relSpeed := math.Hypot(relVelX, relVelY)

	// Closing rate: positive = closing in, negative = separating
	closingRate := 0.0
	if dist > 0.1 {
		closingRate = (dx*relVelX + dy*relVelY) / dist
	}

	// === PHASE 2: Compute intercept point ===

	// Lead the target based on current velocities
	leadTime := math.Min(dist/300.0, npcInterceptLead) // scale lead time with distance
	if leadTime < 0.1 {
		leadTime = 0.1
	}

	// Predict where player will be
	interceptX := player.pos.x + player.vel.x*leadTime
	interceptY := player.pos.y + player.vel.y*leadTime

	// Direction to intercept point
	idx := interceptX - npc.pos.x
	idy := interceptY - npc.pos.y
	interceptDist := math.Hypot(idx, idy)

	// === PHASE 3: Decide behavior mode ===

	// Calculate stopping distance at current closing rate
	// Using physics: d = v²/(2a) where a is our thrust deceleration
	brakingAccel := thrustAccel * npcThrustFactor
	stoppingDist := 0.0
	if closingRate > 0 {
		stoppingDist = (closingRate * closingRate) / (2 * brakingAccel)
	}

	// How far until we reach desired distance?
	distToTarget := dist - npcDesiredDist

	// Determine what we should be doing
	shouldBrake := false
	shouldThrust := false
	shouldMatchVel := false

	if dist < npcDesiredDist*1.2 && relSpeed < 50 {
		// Close and low relative speed - match player velocity (formation keep)
		shouldMatchVel = true
	} else if closingRate > 20 && stoppingDist > distToTarget*0.7 {
		// We're closing AND would overshoot - brake
		// The 0.7 factor gives some margin for the turn time
		shouldBrake = true
	} else if dist > npcDesiredDist*0.5 {
		// Not close enough - pursue
		shouldThrust = true
	}

	// === PHASE 4: Calculate target heading ===

	var targetAngle float64
	if shouldBrake {
		// Point retrograde (opposite to our velocity relative to player)
		if relSpeed > 5 {
			targetAngle = math.Atan2(-relVelX, relVelY)
		} else {
			// Low relative speed - just face the player
			targetAngle = math.Atan2(dx, -dy)
		}
	} else {
		// Point toward intercept
		if interceptDist > 1 {
			targetAngle = math.Atan2(idx, -idy)
		} else {
			targetAngle = npc.angle // hold current heading
		}
	}

	angleDiff := normalizeAngle(targetAngle - npc.angle)

	// === PHASE 5: Smart angular control ===

	// Predict where we'll end up if we start braking angular velocity now
	stoppingAngle := predictAngularStop(npc.angle, npc.angularVel, angularDampingAccel)
	predictedDiff := normalizeAngle(targetAngle - stoppingAngle)

	if math.Abs(angleDiff) > npcAlignThreshold {
		npc.turningThisFrame = true

		// Are we spinning the right way?
		spinningRight := npc.angularVel > 0.1
		spinningLeft := npc.angularVel < -0.1
		needRight := angleDiff > 0
		needLeft := angleDiff < 0

		if spinningRight && needRight {
			// Spinning right, need to go right - check if we should brake
			if math.Abs(predictedDiff) < npcAlignThreshold*2 {
				// We'll overshoot - brake now
				npc.angularVel -= angularDampingAccel * dt
				npc.turnDirection = -1
			} else {
				// Keep accelerating
				npc.angularVel += angularAccel * dt
				npc.turnDirection = 1
			}
		} else if spinningLeft && needLeft {
			// Spinning left, need to go left - check overshoot
			if math.Abs(predictedDiff) < npcAlignThreshold*2 {
				npc.angularVel += angularDampingAccel * dt
				npc.turnDirection = 1
			} else {
				npc.angularVel -= angularAccel * dt
				npc.turnDirection = -1
			}
		} else if spinningRight && needLeft {
			// Wrong direction - brake hard
			npc.angularVel -= angularDampingAccel * dt
			npc.turnDirection = -1
		} else if spinningLeft && needRight {
			// Wrong direction - brake hard
			npc.angularVel += angularDampingAccel * dt
			npc.turnDirection = 1
		} else {
			// Not spinning much - accelerate toward target
			if needRight {
				npc.angularVel += angularAccel * dt
				npc.turnDirection = 1
			} else {
				npc.angularVel -= angularAccel * dt
				npc.turnDirection = -1
			}
		}
	} else {
		// Aligned - dampen remaining spin
		applyAngularDamping(npc, dt)
	}

	// Clamp angular velocity
	npc.angularVel = clamp(npc.angularVel, -maxAngularSpeed, maxAngularSpeed)

	// Update angle
	npc.angle += npc.angularVel * dt

	// === PHASE 6: Apply thrust ===

	forwardX := math.Sin(npc.angle)
	forwardY := -math.Cos(npc.angle)

	if shouldMatchVel {
		// Velocity matching mode - smoothly match player velocity
		velDiffX := player.vel.x - npc.vel.x
		velDiffY := player.vel.y - npc.vel.y
		velDiffMag := math.Hypot(velDiffX, velDiffY)

		if velDiffMag > 1 {
			// Apply thrust in direction of velocity difference, capped by our thrust
			maxDelta := brakingAccel * dt
			if velDiffMag < maxDelta {
				// Can reach target velocity this frame
				npc.vel.x = player.vel.x
				npc.vel.y = player.vel.y
			} else {
				// Apply thrust toward matching velocity
				npc.vel.x += (velDiffX / velDiffMag) * maxDelta
				npc.vel.y += (velDiffY / velDiffMag) * maxDelta
			}
			npc.thrustThisFrame = true
		}
	} else if shouldBrake {
		// Braking - only thrust if aligned with retrograde
		if math.Abs(angleDiff) < npcBrakeAlignThresh {
			thrust := thrustAccel * npcThrustFactor * dt
			npc.vel.x += forwardX * thrust
			npc.vel.y += forwardY * thrust
			npc.thrustThisFrame = true
		}
	} else if shouldThrust {
		// Pursuit - thrust if reasonably aligned with target
		if math.Abs(angleDiff) < math.Pi/3 {
			thrust := thrustAccel * npcThrustFactor * dt
			npc.vel.x += forwardX * thrust
			npc.vel.y += forwardY * thrust
			npc.thrustThisFrame = true
		}
	}

	// Update position
	npc.pos.x += npc.vel.x * dt
	npc.pos.y += npc.vel.y * dt
}

// predictAngularStop predicts the angle when angular velocity reaches zero
func predictAngularStop(angle, angVel, decel float64) float64 {
	if math.Abs(angVel) < 0.01 {
		return angle
	}
	// Time to stop: t = |v| / a
	// Distance traveled: d = v*t - 0.5*a*t^2 = v^2 / (2*a)
	stopDist := (angVel * math.Abs(angVel)) / (2 * decel)
	return angle + stopDist
}

// applyAngularDamping reduces angular velocity toward zero
func applyAngularDamping(ship *Ship, dt float64) {
	if ship.angularVel > 0.01 {
		ship.angularVel -= angularDampingAccel * dt * 0.5
		if ship.angularVel < 0 {
			ship.angularVel = 0
		}
	} else if ship.angularVel < -0.01 {
		ship.angularVel += angularDampingAccel * dt * 0.5
		if ship.angularVel > 0 {
			ship.angularVel = 0
		}
	}
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
