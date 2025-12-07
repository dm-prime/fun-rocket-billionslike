package main

import (
	"fmt"
	"math"
)

// NPCState represents the current behavior state of an NPC
type NPCState int

const (
	NPCStateIdle NPCState = iota
	NPCStatePursue
	NPCStateApproach
	NPCStateHold
	NPCStateLost
)

// NPCStateTransition defines the conditions for transitioning between states
type NPCStateTransition struct {
	fromState   NPCState
	toState     NPCState
	condition   func(dist, closingSpeed, relSpeed, currentSpeed float64) bool
	description string
}

// NPC behavior constants
const (
	npcApproachDist      = 500.0       // distance where we switch to approach mode
	npcHoldDistMin       = 60.0        // minimum hold distance
	npcHoldDistMax       = 100.0       // maximum hold distance
	npcPursueSpeedMult   = 1.1         // pursue speed multiplier relative to player speed
	npcApproachSpeedMult = 0.8         // approach speed multiplier relative to player speed
	npcHoldSpeedMult     = 0.5         // hold speed multiplier relative to player speed
	npcAlignThreshold    = math.Pi / 6 // 30 degrees - alignment threshold for thrusting
	npcMinSpeedToThrust  = 10.0        // minimum speed before we start thrusting
)

// getNPCState gets the current state for an NPC
func (g *Game) getNPCState(entityID EntityID) NPCState {
	if g.npcStates == nil {
		g.npcStates = make(map[EntityID]NPCState)
	}
	if state, ok := g.npcStates[entityID]; ok {
		return state
	}
	return NPCStateIdle
}

// setNPCState sets the state for an NPC
func (g *Game) setNPCState(entityID EntityID, state NPCState) {
	if g.npcStates == nil {
		g.npcStates = make(map[EntityID]NPCState)
	}
	g.npcStates[entityID] = state
}

// getNPCStateString returns a human-readable string for the NPC state
func (g *Game) getNPCStateString(state NPCState) string {
	switch state {
	case NPCStateIdle:
		return "IDLE"
	case NPCStatePursue:
		return "PURSUE"
	case NPCStateApproach:
		return "APPROACH"
	case NPCStateHold:
		return "HOLD"
	case NPCStateLost:
		return "LOST"
	default:
		return "UNKNOWN"
	}
}

// updateNPCStateMachine generates ShipInput for NPC based on state machine
func (g *Game) updateNPCStateMachine(npc *Ship, player *Ship, dt float64) ShipInput {
	// Calculate situation
	dx := player.pos.x - npc.pos.x
	dy := player.pos.y - npc.pos.y
	dist := math.Hypot(dx, dy)

	relVelX := npc.vel.x - player.vel.x
	relVelY := npc.vel.y - player.vel.y
	relSpeed := math.Hypot(relVelX, relVelY)

	closingSpeed := 0.0
	if dist > 1 {
		closingSpeed = (dx*relVelX + dy*relVelY) / dist
	}

	currentSpeed := math.Hypot(npc.vel.x, npc.vel.y)

	// Get NPC entity ID
	npcID := npc.ID()

	// Get current state
	currentState := g.getNPCState(npcID)

	// Determine next state using transition table
	nextState := g.determineNextState(currentState, dist, closingSpeed, relSpeed, currentSpeed)

	// Validate transition
	if nextState == currentState {
		// No transition, stay in current state
	} else {
		// State transition occurred
		if !g.isValidTransition(currentState, nextState, dist, closingSpeed, relSpeed, currentSpeed) {
			// Log the invalid transition instead of panicking
			fmt.Printf("WARNING: Invalid state transition: %s -> %s (dist=%.1f, closingSpeed=%.1f, relSpeed=%.1f, currentSpeed=%.1f)\n",
				g.getNPCStateString(currentState), g.getNPCStateString(nextState), dist, closingSpeed, relSpeed, currentSpeed)
			// Keep current state instead of crashing
		} else {
			g.setNPCState(npcID, nextState)
			currentState = nextState
		}
	}

	// Execute state behavior and generate input
	switch currentState {
	case NPCStateLost:
		return g.executeLostState(npc, player, dt, dx, dy, dist)
	case NPCStatePursue:
		return g.executePursueState(npc, player, dt, dx, dy, dist)
	case NPCStateApproach:
		return g.executeApproachState(npc, player, dt, dx, dy, dist, closingSpeed)
	case NPCStateHold:
		return g.executeHoldState(npc, player, dt, dx, dy, dist, closingSpeed)
	case NPCStateIdle:
		return g.executeIdleState(npc, player, dt, dx, dy, dist)
	default:
		fmt.Printf("WARNING: Unknown NPC state: %d, defaulting to Idle\n", currentState)
		return g.executeIdleState(npc, player, dt, dx, dy, dist)
	}
}

// determineNextState determines the next state based on current state and conditions
func (g *Game) determineNextState(currentState NPCState, dist, closingSpeed, relSpeed, currentSpeed float64) NPCState {
	// Define all valid transitions with explicit conditions
	transitions := g.getStateTransitions()

	// Check each transition from current state
	for _, trans := range transitions {
		if trans.fromState == currentState {
			if trans.condition(dist, closingSpeed, relSpeed, currentSpeed) {
				return trans.toState
			}
		}
	}

	// No valid transition found - stay in current state
	return currentState
}

// isValidTransition checks if a transition is valid according to the transition table
func (g *Game) isValidTransition(from, to NPCState, dist, closingSpeed, relSpeed, currentSpeed float64) bool {
	transitions := g.getStateTransitions()

	for _, trans := range transitions {
		if trans.fromState == from && trans.toState == to {
			// This transition exists in the table
			// Check if conditions are met (or if it's a forced transition)
			return true
		}
	}

	// Transition not found in table - invalid!
	return false
}

// getStateTransitions returns the complete state transition table
func (g *Game) getStateTransitions() []NPCStateTransition {
	return []NPCStateTransition{
		// From LOST state
		{
			fromState: NPCStateLost,
			toState:   NPCStatePursue,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist <= npcReacquireDist
			},
			description: "Lost -> Pursue: Player within reacquire distance",
		},

		// From PURSUE state
		{
			fromState: NPCStatePursue,
			toState:   NPCStateLost,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist > npcReacquireDist
			},
			description: "Pursue -> Lost: Player beyond reacquire distance",
		},
		{
			fromState: NPCStatePursue,
			toState:   NPCStateApproach,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist <= npcApproachDist && dist > npcDesiredDist
			},
			description: "Pursue -> Approach: Within approach distance but beyond desired",
		},
		{
			fromState: NPCStatePursue,
			toState:   NPCStateHold,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist >= npcHoldDistMin && dist <= npcHoldDistMax
			},
			description: "Pursue -> Hold: Within hold distance range",
		},

		// From APPROACH state
		{
			fromState: NPCStateApproach,
			toState:   NPCStateLost,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist > npcReacquireDist
			},
			description: "Approach -> Lost: Player beyond reacquire distance",
		},
		{
			fromState: NPCStateApproach,
			toState:   NPCStatePursue,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist > npcApproachDist
			},
			description: "Approach -> Pursue: Beyond approach distance",
		},
		{
			fromState: NPCStateApproach,
			toState:   NPCStateHold,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist >= npcHoldDistMin && dist <= npcHoldDistMax && math.Abs(closingSpeed) < 20
			},
			description: "Approach -> Hold: Within hold distance and stable",
		},

		// From HOLD state
		{
			fromState: NPCStateHold,
			toState:   NPCStateLost,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist > npcReacquireDist
			},
			description: "Hold -> Lost: Player beyond reacquire distance",
		},
		{
			fromState: NPCStateHold,
			toState:   NPCStateApproach,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return (dist < npcHoldDistMin || dist > npcHoldDistMax) && dist <= npcApproachDist
			},
			description: "Hold -> Approach: Outside hold range but within approach distance",
		},
		{
			fromState: NPCStateHold,
			toState:   NPCStatePursue,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist > npcApproachDist
			},
			description: "Hold -> Pursue: Beyond approach distance",
		},

		// From IDLE state
		{
			fromState: NPCStateIdle,
			toState:   NPCStateLost,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist > npcReacquireDist
			},
			description: "Idle -> Lost: Player beyond reacquire distance",
		},
		{
			fromState: NPCStateIdle,
			toState:   NPCStatePursue,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist <= npcReacquireDist && dist > npcApproachDist
			},
			description: "Idle -> Pursue: Player within range but far",
		},
		{
			fromState: NPCStateIdle,
			toState:   NPCStateApproach,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist <= npcApproachDist && dist > npcDesiredDist
			},
			description: "Idle -> Approach: Player within approach distance",
		},
		{
			fromState: NPCStateIdle,
			toState:   NPCStateHold,
			condition: func(dist, closingSpeed, relSpeed, currentSpeed float64) bool {
				return dist >= npcHoldDistMin && dist <= npcHoldDistMax
			},
			description: "Idle -> Hold: Player within hold distance",
		},
	}
}

// executeLostState handles behavior when player is lost
func (g *Game) executeLostState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64) ShipInput {
	// Point directly at player and burn hard - match player speed
	targetAngle := math.Atan2(dx, -dy)
	return g.generateInputForAngle(npc, targetAngle, true, dt)
}

// executePursueState handles aggressive pursuit
func (g *Game) executePursueState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64) ShipInput {
	// Lead the target for intercept - match player speed
	leadTime := clamp(dist/250.0, 0.1, 1.5)
	targetX := player.pos.x + player.vel.x*leadTime
	targetY := player.pos.y + player.vel.y*leadTime
	tdx := targetX - npc.pos.x
	tdy := targetY - npc.pos.y
	targetAngle := math.Atan2(tdx, -tdy)

	return g.generateInputForAngle(npc, targetAngle, true, dt)
}

// executeApproachState handles careful approach to maintain distance
func (g *Game) executeApproachState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64, closingSpeed float64) ShipInput {
	currentSpeed := math.Hypot(npc.vel.x, npc.vel.y)

	// Calculate stopping distance
	stoppingDist := 0.0
	if closingSpeed > 0 {
		stoppingDist = (closingSpeed * closingSpeed) / (2 * thrustAccel)
	}

	distToDesired := dist - npcDesiredDist

	var targetAngle float64
	shouldThrust := false

	if dist < npcDesiredDist && closingSpeed > 10 && stoppingDist > distToDesired*0.7 {
		// Too close and approaching fast - brake!
		relVelX := npc.vel.x - player.vel.x
		relVelY := npc.vel.y - player.vel.y
		if math.Hypot(relVelX, relVelY) > 5 {
			targetAngle = math.Atan2(-relVelX, relVelY)
		} else {
			targetAngle = math.Atan2(-dx, dy) // Point away from player
		}
		shouldThrust = true
	} else if dist > npcDesiredDist {
		// Too far - approach
		leadTime := clamp(dist/300.0, 0.1, 1.0)
		targetX := player.pos.x + player.vel.x*leadTime
		targetY := player.pos.y + player.vel.y*leadTime
		tdx := targetX - npc.pos.x
		tdy := targetY - npc.pos.y
		targetAngle = math.Atan2(tdx, -tdy)

		// Only thrust if not going too fast relative to player
		playerSpeed := math.Hypot(player.vel.x, player.vel.y)
		targetSpeed := playerSpeed * npcApproachSpeedMult
		if currentSpeed < targetSpeed {
			shouldThrust = true
		}
	} else {
		// At desired distance - coast
		targetAngle = math.Atan2(dx, -dy)
		shouldThrust = false
	}

	return g.generateInputForAngle(npc, targetAngle, shouldThrust, dt)
}

// executeHoldState maintains position at desired distance
func (g *Game) executeHoldState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64, closingSpeed float64) ShipInput {
	currentSpeed := math.Hypot(npc.vel.x, npc.vel.y)
	relVelX := npc.vel.x - player.vel.x
	relVelY := npc.vel.y - player.vel.y

	var targetAngle float64
	shouldThrust := false

	// Match player's velocity direction, but maintain distance
	if closingSpeed > 15 {
		// Approaching too fast - brake
		targetAngle = math.Atan2(-relVelX, relVelY)
		shouldThrust = true
	} else if closingSpeed < -10 {
		// Moving away - accelerate toward player
		targetAngle = math.Atan2(dx, -dy)
		playerSpeed := math.Hypot(player.vel.x, player.vel.y)
		targetSpeed := playerSpeed * npcHoldSpeedMult
		if currentSpeed < targetSpeed {
			shouldThrust = true
		}
	} else {
		// Good position - match player's heading roughly
		targetAngle = player.angle
		// Only thrust if speed is very low
		if currentSpeed < npcMinSpeedToThrust {
			shouldThrust = true
		}
	}

	return g.generateInputForAngle(npc, targetAngle, shouldThrust, dt)
}

// executeIdleState handles idle/patrol behavior
func (g *Game) executeIdleState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64) ShipInput {
	// Simple: turn toward player but don't thrust much
	targetAngle := math.Atan2(dx, -dy)
	currentSpeed := math.Hypot(npc.vel.x, npc.vel.y)
	shouldThrust := currentSpeed < npcMinSpeedToThrust
	return g.generateInputForAngle(npc, targetAngle, shouldThrust, dt)
}

// generateInputForAngle generates ShipInput to turn toward a target angle and optionally thrust
func (g *Game) generateInputForAngle(npc *Ship, targetAngle float64, shouldThrust bool, dt float64) ShipInput {
	input := ShipInput{}
	angleDiff := normalizeAngle(targetAngle - npc.angle)

	// PID-lite controller for turning
	desiredAngVel := clamp(angleDiff*3.2, -maxAngularSpeed*0.6, maxAngularSpeed*0.6)
	if math.Abs(angleDiff) < 0.25 && math.Abs(desiredAngVel) < 0.3 {
		desiredAngVel = 0
	}

	// Determine turn direction based on desired angular velocity
	if desiredAngVel > 0.01 {
		input.TurnRight = true
	} else if desiredAngVel < -0.01 {
		input.TurnLeft = true
	}

	// Thrust if aligned and should thrust
	if shouldThrust && math.Abs(angleDiff) < npcAlignThreshold {
		input.ThrustForward = true
	}

	return input
}
