package main

import "math"

// PhysicsState represents the kinematic state of an entity for physics simulation.
// This abstraction allows the same physics code to be used for both real-time updates
// and predictive trail calculations, eliminating code duplication.
type PhysicsState struct {
	pos        vec2
	vel        vec2
	angle      float64
	angularVel float64
}

// NewPhysicsStateFromShip creates a physics state snapshot from a ship's current state.
// This is used to initialize simulation without modifying the original ship.
func NewPhysicsStateFromShip(ship *Ship) PhysicsState {
	return PhysicsState{
		pos:        ship.pos,
		vel:        ship.vel,
		angle:      ship.angle,
		angularVel: ship.angularVel,
	}
}

// ApplyToShip writes the physics state back to a ship, updating its position and velocity.
// This is used after simulation to commit the changes to the actual entity.
func (ps *PhysicsState) ApplyToShip(ship *Ship) {
	ship.pos = ps.pos
	ship.vel = ps.vel
	ship.angle = ps.angle
	ship.angularVel = ps.angularVel
}

// simulatePhysicsStep simulates one physics timestep with the given input.
// This is the unified physics simulation core used by both:
// - Real-time ship updates (updatePhysics)
// - Predictive trail calculations (predictFuturePath)
// By centralizing the physics logic here, we ensure consistency and reduce maintenance burden.
func simulatePhysicsStep(state *PhysicsState, input ShipInput, dt float64) {
	// Apply angular acceleration based on input
	if input.TurnLeft {
		state.angularVel -= angularAccel * dt
	}
	if input.TurnRight {
		state.angularVel += angularAccel * dt
	}

	// Clamp angular velocity to max speed
	if state.angularVel > maxAngularSpeed {
		state.angularVel = maxAngularSpeed
	}
	if state.angularVel < -maxAngularSpeed {
		state.angularVel = -maxAngularSpeed
	}

	// Automatically apply angular damping when no turn input
	if !input.TurnLeft && !input.TurnRight && math.Abs(state.angularVel) > angularVelThreshold {
		if state.angularVel > 0 {
			state.angularVel -= angularDampingAccel * dt * autoDampingMultiplier
			if state.angularVel < 0 {
				state.angularVel = 0
			}
		} else {
			state.angularVel += angularDampingAccel * dt * autoDampingMultiplier
			if state.angularVel > 0 {
				state.angularVel = 0
			}
		}
	}

	// Update ship angle based on angular velocity
	state.angle += state.angularVel * dt

	forwardX := math.Sin(state.angle)
	forwardY := -math.Cos(state.angle)

	// Apply forward thrust
	if input.ThrustForward {
		state.vel.x += forwardX * thrustAccel * dt
		state.vel.y += forwardY * thrustAccel * dt
	}

	// Apply reverse thrust (negative main engine thrust)
	if input.ReverseThrust {
		state.vel.x -= forwardX * thrustAccel * dt
		state.vel.y -= forwardY * thrustAccel * dt
	}

	// Update position
	state.pos.x += state.vel.x * dt
	state.pos.y += state.vel.y * dt
}
