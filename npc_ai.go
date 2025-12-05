package main

import "math"

// NPCState represents the current behavior state of an NPC
type NPCState int

const (
	NPCStateIdle NPCState = iota
	NPCStatePursue
	NPCStateApproach
	NPCStateHold
	NPCStateLost
)

// NPCStateData holds state-specific data for NPCs
type NPCStateData struct {
	state NPCState
	timer float64
}

// NPC behavior constants
const (
	npcApproachDist     = 150.0 // distance where we switch to approach mode
	npcHoldDistMin      = 60.0  // minimum hold distance
	npcHoldDistMax      = 100.0 // maximum hold distance
	npcPursueSpeed      = 180.0 // speed when pursuing
	npcApproachSpeed    = 120.0 // speed when approaching
	npcHoldSpeed        = 80.0  // speed when holding position
	npcAlignThreshold   = math.Pi / 6 // 30 degrees - alignment threshold for thrusting
	npcMinSpeedToThrust = 10.0  // minimum speed before we start thrusting
)

// getNPCStateData gets or initializes state data for an NPC
func (g *Game) getNPCStateData(shipIndex int) NPCStateData {
	// For now, we'll store state in a map. In a real implementation, you might want
	// to add this to the Ship struct or Game struct
	// For simplicity, we'll use a simple approach and track state per update
	return NPCStateData{state: int(NPCStateIdle), timer: 0}
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

	// Determine current state based on situation
	currentState := g.determineNPCState(dist, closingSpeed, relSpeed)
	
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
	default: // NPCStateIdle
		g.executeIdleState(npc, player, dt, dx, dy, dist)
	}

	// Update position
	npc.pos.x += npc.vel.x * dt
	npc.pos.y += npc.vel.y * dt
}

// determineNPCState decides which state the NPC should be in
func (g *Game) determineNPCState(dist, closingSpeed, relSpeed float64) NPCState {
	if dist > npcReacquireDist {
		return NPCStateLost
	}
	
	if dist > npcApproachDist {
		return NPCStatePursue
	}
	
	if dist > npcDesiredDist {
		return NPCStateApproach
	}
	
	if dist >= npcHoldDistMin && dist <= npcHoldDistMax {
		return NPCStateHold
	}
	
	// Too close, need to back off
	if dist < npcHoldDistMin {
		return NPCStateApproach // Use approach logic to back off
	}
	
	return NPCStatePursue
}

// executeLostState handles behavior when player is lost
func (g *Game) executeLostState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64) {
	// Point directly at player and burn hard
	targetAngle := math.Atan2(dx, -dy)
	g.turnTowardAngle(npc, targetAngle, dt)
	g.thrustIfAligned(npc, targetAngle, npcPursueSpeed, dt)
}

// executePursueState handles aggressive pursuit
func (g *Game) executePursueState(npc *Ship, player *Ship, dt float64, dx, dy, dist float64) {
	// Lead the target for intercept
	leadTime := clamp(dist/250.0, 0.1, 1.5)
	targetX := player.pos.x + player.vel.x*leadTime
	targetY := player.pos.y + player.vel.y*leadTime
	tdx := targetX - npc.pos.x
	tdy := targetY - npc.pos.y
	targetAngle := math.Atan2(tdx, -tdy)
	
	g.turnTowardAngle(npc, targetAngle, dt)
	g.thrustIfAligned(npc, targetAngle, npcPursueSpeed, dt)
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
		
		// Only thrust if not going too fast
		if currentSpeed < npcApproachSpeed {
			shouldThrust = true
		}
	} else {
		// At desired distance - coast
		targetAngle = math.Atan2(dx, -dy)
		shouldThrust = false
	}
	
	g.turnTowardAngle(npc, targetAngle, dt)
	if shouldThrust {
		g.thrustIfAligned(npc, targetAngle, npcApproachSpeed, dt)
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
		if currentSpeed < npcHoldSpeed {
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
		g.thrustIfAligned(npc, targetAngle, npcHoldSpeed, dt)
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

