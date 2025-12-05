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
func (g *Game) getNPCState(shipIndex int) NPCState {
	if g.npcStates == nil {
		g.npcStates = make(map[int]NPCState)
	}
	if state, ok := g.npcStates[shipIndex]; ok {
		return state
	}
	return NPCStateIdle
}

// setNPCState sets the state for an NPC
func (g *Game) setNPCState(shipIndex int, state NPCState) {
	if g.npcStates == nil {
		g.npcStates = make(map[int]NPCState)
	}
	g.npcStates[shipIndex] = state
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

// updateNPCStateMachine updates NPC behavior using a state machine
func (g *Game) updateNPCStateMachine(npc *Ship, player *Ship, dt float64) {
	npc.thrustThisFrame = false
	npc.turningThisFrame = false
	npc.turnDirection = 0

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

	// Find ship index for state tracking
	shipIndex := -1
	for i := range g.ships {
		if &g.ships[i] == npc {
			shipIndex = i
			break
		}
	}
	if shipIndex < 0 {
		return // Ship not found, skip
	}

	// Get current state
	currentState := g.getNPCState(shipIndex)

	// Determine next state using transition table
	nextState := g.determineNextState(currentState, dist, closingSpeed, relSpeed, currentSpeed)

	// Validate transition
	if nextState == currentState {
		// No transition, stay in current state
	} else {
		// State transition occurred
		if !g.isValidTransition(currentState, nextState, dist, closingSpeed, relSpeed, currentSpeed) {
			panic(fmt.Sprintf("INVALID STATE TRANSITION: %s -> %s (dist=%.1f, closingSpeed=%.1f, relSpeed=%.1f, currentSpeed=%.1f)",
				g.getNPCStateString(currentState), g.getNPCStateString(nextState), dist, closingSpeed, relSpeed, currentSpeed))
		}
		g.setNPCState(shipIndex, nextState)
		currentState = nextState
	}

	// Execute state behavior
	switch currentState {
	case NPCStateLost:
		g.executeLostState(npc, player, dt, dx, dy, dist)
	case NPCStatePursue:
		g.executePursueState(npc, player, dt, dx, dy, dist)
	case NPCStateApproach:
		g.executeApproachState(npc, player, dt, dx, dy, dist, closingSpeed)
	case NPCStateHold:
		g.executeHoldState(npc, player, dt, dx, dy, dist, closingSpeed)
	case NPCStateIdle:
		g.executeIdleState(npc, player, dt, dx, dy, dist)
	default:
		panic(fmt.Sprintf("UNKNOWN STATE: %d", currentState))
	}

	// Update position
	npc.pos.x += npc.vel.x * dt
	npc.pos.y += npc.vel.y * dt
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
func (g *Game) executeLostState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64) {
	// Point directly at player and burn hard - match player speed
	playerSpeed := math.Hypot(player.vel.x, player.vel.y)
	targetSpeed := playerSpeed * npcPursueSpeedMult
	targetAngle := math.Atan2(dx, -dy)
	g.turnTowardAngle(npc, targetAngle, dt)
	g.thrustIfAligned(npc, targetAngle, targetSpeed, dt)
}

// executePursueState handles aggressive pursuit
func (g *Game) executePursueState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64) {
	// Lead the target for intercept - match player speed
	playerSpeed := math.Hypot(player.vel.x, player.vel.y)
	targetSpeed := playerSpeed * npcPursueSpeedMult
	leadTime := clamp(dist/250.0, 0.1, 1.5)
	targetX := player.pos.x + player.vel.x*leadTime
	targetY := player.pos.y + player.vel.y*leadTime
	tdx := targetX - npc.pos.x
	tdy := targetY - npc.pos.y
	targetAngle := math.Atan2(tdx, -tdy)

	g.turnTowardAngle(npc, targetAngle, dt)
	g.thrustIfAligned(npc, targetAngle, targetSpeed, dt)
}

// executeApproachState handles careful approach to maintain distance
func (g *Game) executeApproachState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64, closingSpeed float64) {
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

	g.turnTowardAngle(npc, targetAngle, dt)
	if shouldThrust {
		playerSpeed := math.Hypot(player.vel.x, player.vel.y)
		targetSpeed := playerSpeed * npcApproachSpeedMult
		g.thrustIfAligned(npc, targetAngle, targetSpeed, dt)
	}
}

// executeHoldState maintains position at desired distance
func (g *Game) executeHoldState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64, closingSpeed float64) {
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

	g.turnTowardAngle(npc, targetAngle, dt)
	if shouldThrust {
		playerSpeed := math.Hypot(player.vel.x, player.vel.y)
		targetSpeed := playerSpeed * npcHoldSpeedMult
		g.thrustIfAligned(npc, targetAngle, targetSpeed, dt)
	}
}

// executeIdleState handles idle/patrol behavior
func (g *Game) executeIdleState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64) {
	// Simple: turn toward player but don't thrust much
	targetAngle := math.Atan2(dx, -dy)
	g.turnTowardAngle(npc, targetAngle, dt)

	currentSpeed := math.Hypot(npc.vel.x, npc.vel.y)
	if currentSpeed < npcMinSpeedToThrust {
		g.thrustIfAligned(npc, targetAngle, 50.0, dt)
	}
}

// turnTowardAngle handles turning logic toward a target angle
func (g *Game) turnTowardAngle(npc *Ship, targetAngle float64, dt float64) {
	angleDiff := normalizeAngle(targetAngle - npc.angle)

	// PID-lite controller
	desiredAngVel := clamp(angleDiff*3.2, -maxAngularSpeed*0.6, maxAngularSpeed*0.6)
	if math.Abs(angleDiff) < 0.25 && math.Abs(desiredAngVel) < 0.3 {
		desiredAngVel = 0
	}

	maxStep := angularAccel * dt
	angVelError := desiredAngVel - npc.angularVel
	if math.Abs(angVelError) > 0.0001 {
		delta := clamp(angVelError, -maxStep, maxStep)
		npc.angularVel += delta
		if delta > 0.0001 {
			npc.turningThisFrame = true
			npc.turnDirection = 1
		} else if delta < -0.0001 {
			npc.turningThisFrame = true
			npc.turnDirection = -1
		}
	} else {
		npc.angularVel = desiredAngVel
	}

	// Extra damping when close to target
	if math.Abs(angleDiff) < 0.06 && math.Abs(npc.angularVel) < 0.12 {
		npc.angularVel = 0
		npc.turningThisFrame = false
		npc.turnDirection = 0
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
}

// thrustIfAligned only thrusts if aligned with target angle and speed conditions are met
func (g *Game) thrustIfAligned(npc *Ship, targetAngle float64, maxSpeed float64, dt float64) {
	angleDiff := normalizeAngle(targetAngle - npc.angle)
	currentSpeed := math.Hypot(npc.vel.x, npc.vel.y)

	// Only thrust if aligned and not going too fast
	if math.Abs(angleDiff) < npcAlignThreshold && currentSpeed < maxSpeed {
		forwardX := math.Sin(npc.angle)
		forwardY := -math.Cos(npc.angle)
		npc.vel.x += forwardX * thrustAccel * dt
		npc.vel.y += forwardY * thrustAccel * dt
		npc.thrustThisFrame = true
	}
}
