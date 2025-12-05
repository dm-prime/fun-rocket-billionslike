package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// updatePhysics applies input-driven forces, rotation, and retrograde logic.
func (g *Game) updatePhysics(dt float64) {
	g.thrustThisFrame = false
	g.turningThisFrame = false
	g.turnDirection = 0
	g.dampingAngularSpeed = false

	// Apply angular acceleration based on input
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.shipAngularVel -= angularAccel * dt
		g.turningThisFrame = true
		g.turnDirection = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.shipAngularVel += angularAccel * dt
		g.turningThisFrame = true
		g.turnDirection = 1
	}

	// Clamp angular velocity to max speed
	if g.shipAngularVel > maxAngularSpeed {
		g.shipAngularVel = maxAngularSpeed
	}
	if g.shipAngularVel < -maxAngularSpeed {
		g.shipAngularVel = -maxAngularSpeed
	}

	// Automatically apply angular damping when no turn input (A/D not pressed)
	if !g.turningThisFrame && math.Abs(g.shipAngularVel) > 0.01 {
		// Gradually reduce angular velocity
		if g.shipAngularVel > 0 {
			g.shipAngularVel -= angularDampingAccel * dt * 0.5
			if g.shipAngularVel < 0 {
				g.shipAngularVel = 0
			}
		} else {
			g.shipAngularVel += angularDampingAccel * dt * 0.5
			if g.shipAngularVel > 0 {
				g.shipAngularVel = 0
			}
		}
	}

	// Update ship angle based on angular velocity
	g.shipAngle += g.shipAngularVel * dt

	forwardX := math.Sin(g.shipAngle)
	forwardY := -math.Cos(g.shipAngle)

	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.shipVel.x += forwardX * thrustAccel * dt
		g.shipVel.y += forwardY * thrustAccel * dt
		g.thrustThisFrame = true
	}

	// S key activates retrograde burn mode
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		if !g.retrogradeMode {
			// Entering retrograde mode - calculate the fastest turn direction
			g.retrogradeMode = true
			g.retrogradeTurnDir = g.calculateFastestRetrogradeTurn()
		}
		// Execute retrograde burn maneuver
		if g.retrogradeMode {
			g.executeRetrogradeBurn(dt)
		}
	} else {
		// S key not held - immediately cancel retrograde mode
		if g.retrogradeMode {
			g.retrogradeMode = false
			g.retrogradeTurnDir = 0
		}
	}

	g.shipPos.x += g.shipVel.x * dt
	g.shipPos.y += g.shipVel.y * dt
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
func (g *Game) calculateFastestRetrogradeTurn() float64 {
	speed := math.Hypot(g.shipVel.x, g.shipVel.y)
	if speed < 5.0 {
		return 0
	}

	// Calculate retrograde angle (opposite to velocity)
	// Ship forward is (sin(angle), -cos(angle))
	targetAngle := math.Atan2(-g.shipVel.x, g.shipVel.y)
	angleDiff := normalizeAngle(targetAngle - g.shipAngle)

	// Calculate time for short path vs long path
	shortTime := estimateTurnTime(angleDiff, g.shipAngularVel, angularAccel)

	var longDist float64
	if angleDiff > 0 {
		longDist = angleDiff - 2*math.Pi
	} else {
		longDist = angleDiff + 2*math.Pi
	}
	longTime := estimateTurnTime(longDist, g.shipAngularVel, angularAccel)

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
func (g *Game) executeRetrogradeBurn(dt float64) {
	speed := math.Hypot(g.shipVel.x, g.shipVel.y)

	// Check if velocity is killed
	if speed < 2.0 {
		g.retrogradeMode = false
		g.retrogradeTurnDir = 0
		return
	}

	// Always recalculate target angle each frame based on current velocity
	targetAngle := math.Atan2(-g.shipVel.x, g.shipVel.y)
	angleDiff := normalizeAngle(targetAngle - g.shipAngle)

	// Continuously align against speed - always turn towards retrograde direction
	g.turningThisFrame = true
	g.dampingAngularSpeed = true

	// Determine turn direction based on angle difference
	if math.Abs(angleDiff) > 0.01 { // Small threshold to avoid jitter
		// Determine which direction to turn (shortest path)
		if angleDiff > 0 {
			g.retrogradeTurnDir = 1.0 // turn right
		} else {
			g.retrogradeTurnDir = -1.0 // turn left
		}
		g.turnDirection = g.retrogradeTurnDir

		// Apply angular acceleration to turn towards retrograde
		if g.retrogradeTurnDir > 0 {
			g.shipAngularVel += angularAccel * dt
		} else {
			g.shipAngularVel -= angularAccel * dt
		}
	} else {
		// Very close to alignment - dampen angular velocity to maintain alignment
		if math.Abs(g.shipAngularVel) > 0.01 {
			if g.shipAngularVel > 0 {
				g.shipAngularVel -= angularDampingAccel * dt
				g.turnDirection = -1
			} else {
				g.shipAngularVel += angularDampingAccel * dt
				g.turnDirection = 1
			}
		}
	}

	// Always fire main engine while in retrograde mode (continuous alignment + burn)
	forwardX := math.Sin(g.shipAngle)
	forwardY := -math.Cos(g.shipAngle)
	g.shipVel.x += forwardX * thrustAccel * dt
	g.shipVel.y += forwardY * thrustAccel * dt
	g.thrustThisFrame = true
}
