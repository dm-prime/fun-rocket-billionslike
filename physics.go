package main

import (
	"fmt"
	"math"
	"math/rand"
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

// isOnCollisionCourse checks if the player and rock are on a collision course within the look-ahead time
func (g *Game) isOnCollisionCourse(player *Ship, rock *Ship, lookAheadTime float64) bool {
	// Get current player input to predict future path
	playerInput := getPlayerInput()

	// Predict player's future positions
	playerPositions := g.predictFuturePath(player, playerInput)

	// Rock moves linearly (no acceleration)
	// Check if any predicted player position intersects with rock's path
	dt := predictiveTrailUpdateRate
	steps := int(lookAheadTime / dt)
	if steps > len(playerPositions)-1 {
		steps = len(playerPositions) - 1
	}

	collisionRadius := shipCollisionRadius + rockRadius

	for i := 0; i < steps; i++ {
		// Predict rock position at this time
		t := float64(i) * dt
		rockFuturePos := vec2{
			x: rock.pos.x + rock.vel.x*t,
			y: rock.pos.y + rock.vel.y*t,
		}

		// Check distance between predicted positions
		dx := playerPositions[i].x - rockFuturePos.x
		dy := playerPositions[i].y - rockFuturePos.y
		dist := math.Hypot(dx, dy)

		if dist < collisionRadius {
			return true
		}
	}

	return false
}

// checkCollision checks if two ships are currently colliding
func (g *Game) checkCollision(ship1 *Ship, ship2 *Ship) bool {
	dx := ship1.pos.x - ship2.pos.x
	dy := ship1.pos.y - ship2.pos.y
	dist := math.Hypot(dx, dy)
	
	var radius1, radius2 float64
	if g.isRock(ship1) {
		radius1 = rockRadius
	} else {
		radius1 = shipCollisionRadius
	}
	if g.isRock(ship2) {
		radius2 = rockRadius
	} else {
		radius2 = shipCollisionRadius
	}
	
	collisionRadius := radius1 + radius2
	return dist < collisionRadius
}

// checkBulletCollision checks if a bullet is colliding with a ship
func (g *Game) checkBulletCollision(bullet *Bullet, ship *Ship) bool {
	dx := bullet.pos.x - ship.pos.x
	dy := bullet.pos.y - ship.pos.y
	dist := math.Hypot(dx, dy)
	
	var shipRadius float64
	if g.isRock(ship) {
		shipRadius = rockRadius
	} else {
		shipRadius = shipCollisionRadius
	}
	
	collisionRadius := bulletRadius + shipRadius
	return dist < collisionRadius
}

// checkBulletCollisions checks all bullets against all ships and applies damage
func (g *Game) checkBulletCollisions(dt float64) {
	bulletsToRemove := make([]int, 0)
	
	for i := range g.bullets {
		bullet := &g.bullets[i]
		
		// Skip if bullet was already marked for removal
		shouldRemove := false
		for _, idx := range bulletsToRemove {
			if idx == i {
				shouldRemove = true
				break
			}
		}
		if shouldRemove {
			continue
		}
		
		// Check collision with all ships
		for j := range g.ships {
			target := &g.ships[j]
			
			// Skip if target is dead
			if target.health <= 0 {
				continue
			}
			
			// Skip if bullet was fired by this ship
			if bullet.shipIdx == j {
				continue
			}
			
			// Skip if target is allied with bullet's faction
			if g.areAllied(bullet.faction, target.faction) {
				continue
			}
			
			// Check collision
			if g.checkBulletCollision(bullet, target) {
				// Apply damage
				target.health -= bullet.damage
				if target.health < 0 {
					target.health = 0
				}
				
				// Remove bullet on hit
				bulletsToRemove = append(bulletsToRemove, i)
				break // Bullet can only hit one target
			}
		}
	}
	
	// Remove bullets that hit targets (in reverse order to maintain indices)
	for i := len(bulletsToRemove) - 1; i >= 0; i-- {
		idx := bulletsToRemove[i]
		if idx < len(g.bullets) {
			// Remove by swapping with last element
			lastIdx := len(g.bullets) - 1
			if idx != lastIdx {
				g.bullets[idx] = g.bullets[lastIdx]
			}
			g.bullets = g.bullets[:lastIdx]
		}
	}
}

// checkShipCollisions checks all ships against each other and applies collision damage
func (g *Game) checkShipCollisions(dt float64) {
	// Track collisions to avoid double-processing
	collisionPairs := make(map[string]bool)
	
	for i := range g.ships {
		ship1 := &g.ships[i]
		
		// Skip if dead
		if ship1.health <= 0 {
			continue
		}
		
		for j := i + 1; j < len(g.ships); j++ {
			ship2 := &g.ships[j]
			
			// Skip if dead
			if ship2.health <= 0 {
				continue
			}
			
			// Create unique key for collision pair
			key := ""
			if i < j {
				key = fmt.Sprintf("%d-%d", i, j)
			} else {
				key = fmt.Sprintf("%d-%d", j, i)
			}
			
			// Skip if already processed
			if collisionPairs[key] {
				continue
			}
			
			// Check collision
			if g.checkCollision(ship1, ship2) {
				collisionPairs[key] = true
				
				// Calculate collision damage
				var damage float64
				if g.isRock(ship1) || g.isRock(ship2) {
					// Rock collisions deal more damage
					damage = rockCollisionDamage
				} else {
					// Ship-ship collisions
					damage = shipCollisionDamage
				}
				
				// Apply damage to both ships
				ship1.health -= damage
				if ship1.health < 0 {
					ship1.health = 0
				}
				
				ship2.health -= damage
				if ship2.health < 0 {
					ship2.health = 0
				}
				
				// Apply some velocity change on collision (simple bounce)
				dx := ship2.pos.x - ship1.pos.x
				dy := ship2.pos.y - ship1.pos.y
				dist := math.Hypot(dx, dy)
				if dist > 0.1 {
					// Normalize direction
					dx /= dist
					dy /= dist
					
					// Apply impulse (push ships apart)
					impulse := 50.0 * dt
					ship1.vel.x -= dx * impulse
					ship1.vel.y -= dy * impulse
					ship2.vel.x += dx * impulse
					ship2.vel.y += dy * impulse
				}
			}
		}
	}
}

// removeDeadShips removes ships with health <= 0 (except player - handle separately)
func (g *Game) removeDeadShips() {
	// Build map of old index to new index
	indexMap := make(map[int]int)
	validShips := make([]Ship, 0)
	newIndex := 0
	
	for i := range g.ships {
		ship := &g.ships[i]
		
		// Keep player even if dead (for game over handling later)
		if ship.isPlayer {
			indexMap[i] = newIndex
			validShips = append(validShips, *ship)
			newIndex++
			continue
		}
		
		// Remove dead ships
		if ship.health <= 0 {
			continue
		}
		
		indexMap[i] = newIndex
		validShips = append(validShips, *ship)
		newIndex++
	}
	
	// Only update if ships were removed
	if len(validShips) != len(g.ships) {
		// Update player index
		newPlayerIndex := -1
		for i := range validShips {
			if validShips[i].isPlayer {
				newPlayerIndex = i
				break
			}
		}
		if newPlayerIndex >= 0 {
			g.playerIndex = newPlayerIndex
		}
		
		// Update bullet ship indices using index map
		for i := range g.bullets {
			bullet := &g.bullets[i]
			if bullet.shipIdx >= 0 {
				if newIdx, ok := indexMap[bullet.shipIdx]; ok {
					bullet.shipIdx = newIdx
				} else {
					bullet.shipIdx = -1 // Ship was removed
				}
			}
			
			// Update homing missile target indices
			if bullet.isHoming && bullet.targetIdx >= 0 {
				if newIdx, ok := indexMap[bullet.targetIdx]; ok {
					bullet.targetIdx = newIdx
				} else {
					bullet.isHoming = false // Target is dead
					bullet.targetIdx = -1
				}
			}
		}
		
		// Clean up NPC state maps using index map
		newNPCStates := make(map[int]NPCState)
		newNPCInputs := make(map[int]ShipInput)
		for oldIdx, newIdx := range indexMap {
			if !validShips[newIdx].isPlayer {
				if oldState, ok := g.npcStates[oldIdx]; ok {
					newNPCStates[newIdx] = oldState
				}
				if oldInput, ok := g.npcInputs[oldIdx]; ok {
					newNPCInputs[newIdx] = oldInput
				}
			}
		}
		g.npcStates = newNPCStates
		g.npcInputs = newNPCInputs
		
		g.ships = validShips
	}
}

// isNearPlayerPath checks if a position is near the player's predicted path
func (g *Game) isNearPlayerPath(pos vec2, player *Ship) bool {
	playerInput := getPlayerInput()
	playerPath := g.predictFuturePath(player, playerInput)

	// Check distance to any point on the predicted path
	for _, pathPos := range playerPath {
		dx := pos.x - pathPos.x
		dy := pos.y - pathPos.y
		dist := math.Hypot(dx, dy)
		if dist < rockPathDistance {
			return true
		}
	}

	// Also check distance to current player position
	dx := pos.x - player.pos.x
	dy := pos.y - player.pos.y
	dist := math.Hypot(dx, dy)
	return dist < rockPathDistance
}

// spawnRockNearPath spawns a rock near the player's predicted path
func (g *Game) spawnRockNearPath(player *Ship) {
	playerInput := getPlayerInput()
	playerPath := g.predictFuturePath(player, playerInput)

	if len(playerPath) == 0 {
		return
	}

	// Pick a random point along the predicted path (prefer further ahead)
	pathIndex := rand.Intn(len(playerPath))
	if len(playerPath) > 5 {
		// Bias towards spawning ahead (last 2/3 of path)
		startIdx := len(playerPath) * 2 / 3
		pathIndex = startIdx + rand.Intn(len(playerPath)-startIdx)
	}

	spawnPos := playerPath[pathIndex]

	// Add some random offset perpendicular to the path direction
	// Get direction from previous point (or use player velocity if at start)
	var dir vec2
	if pathIndex > 0 {
		dir = vec2{
			x: playerPath[pathIndex].x - playerPath[pathIndex-1].x,
			y: playerPath[pathIndex].y - playerPath[pathIndex-1].y,
		}
	} else {
		dir = vec2{x: player.vel.x, y: player.vel.y}
	}

	// Normalize direction
	dirLen := math.Hypot(dir.x, dir.y)
	if dirLen < 0.1 {
		// Use a default direction if velocity is too small
		dir = vec2{x: math.Sin(player.angle), y: -math.Cos(player.angle)}
		dirLen = 1.0
	}
	dir.x /= dirLen
	dir.y /= dirLen

	// Perpendicular vector (rotate 90 degrees)
	perp := vec2{x: -dir.y, y: dir.x}

	// Random offset perpendicular to path (within rockPathDistance)
	offsetDist := (rand.Float64() - 0.5) * rockPathDistance * 0.8
	spawnPos.x += perp.x * offsetDist
	spawnPos.y += perp.y * offsetDist

	// Also add some forward/backward offset
	forwardOffset := (rand.Float64() - 0.5) * rockPathDistance * 0.5
	spawnPos.x += dir.x * forwardOffset
	spawnPos.y += dir.y * forwardOffset

	// Check if spawn position is outside view range (only spawn rocks outside visible area)
	dx := spawnPos.x - player.pos.x
	dy := spawnPos.y - player.pos.y
	distFromPlayer := math.Hypot(dx, dy)

	// Only spawn if outside minimum spawn distance (outside view range)
	if distFromPlayer < rockMinSpawnDistance {
		return // Don't spawn this rock, it's too close to player
	}

	// Create rock with small drift velocity
	rock := Ship{
		pos: spawnPos,
		vel: vec2{
			x: (rand.Float64() - 0.5) * 50.0, // Random drift velocity between -25 and 25 px/s
			y: (rand.Float64() - 0.5) * 50.0,
		},
		angle:      rand.Float64() * 2 * math.Pi, // Random initial rotation
		angularVel: 0,                            // No rotation
		health:     100,
		faction:    "Rocks",
		isPlayer:   false,
	}

	g.ships = append(g.ships, rock)
}

// manageRocks handles spawning and despawning rocks based on player's path
func (g *Game) manageRocks(player *Ship, dt float64) {
	// Count current rocks and find ones to remove
	currentRockCount := 0
	rocksToRemove := make([]int, 0)

	for i := range g.ships {
		if g.isRock(&g.ships[i]) {
			currentRockCount++
			rock := &g.ships[i]

			// Check distance to player
			dx := rock.pos.x - player.pos.x
			dy := rock.pos.y - player.pos.y
			distToPlayer := math.Hypot(dx, dy)

			// Mark for removal if far from player OR far from player's path
			if distToPlayer > rockDespawnDistance || !g.isNearPlayerPath(rock.pos, player) {
				rocksToRemove = append(rocksToRemove, i)
			}
		}
	}

	// Remove rocks in reverse order to maintain indices
	for i := len(rocksToRemove) - 1; i >= 0; i-- {
		rockIdx := rocksToRemove[i]
		// Remove by swapping with last element and truncating
		lastIdx := len(g.ships) - 1
		if rockIdx != lastIdx {
			g.ships[rockIdx] = g.ships[lastIdx]
		}
		g.ships = g.ships[:lastIdx]
		currentRockCount--
	}

	// Spawn new rocks if below target count
	g.rockSpawnTimer += dt
	if currentRockCount < rockCount && g.rockSpawnTimer >= rockSpawnInterval {
		g.spawnRockNearPath(player)
		g.rockSpawnTimer = 0
	}
}
