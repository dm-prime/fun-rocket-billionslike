package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// initTurretPoints initializes turret points for a ship
func (g *Game) initTurretPoints(ship *Ship) {
	// Add two turret points on the sides of the ship
	ship.turretPoints = []vec2{
		{turretLeftX, turretLeftY},   // left turret
		{turretRightX, turretRightY}, // right turret
	}
	ship.lastFireTime = 0
}

// spawnBullet creates a bullet from a turret point
func (g *Game) spawnBullet(ship *Ship, turretIdx int, targetAngle float64) {
	g.spawnBulletWithTarget(ship, turretIdx, targetAngle, nil, false)
}

// spawnBulletWithTarget creates a bullet or homing missile from a turret point
func (g *Game) spawnBulletWithTarget(ship *Ship, turretIdx int, targetAngle float64, target Entity, isHoming bool) {
	if turretIdx < 0 || turretIdx >= len(ship.turretPoints) {
		return
	}

	// Get turret position in world space
	turretLocal := ship.turretPoints[turretIdx]
	turretWorld := rotatePoint(turretLocal, ship.angle)
	turretWorld.x += ship.pos.x
	turretWorld.y += ship.pos.y

	// Get ship ID
	shipID := ship.ID()

	// Get target ID if homing
	var targetID EntityID = InvalidEntityID
	if isHoming && target != nil {
		targetID = target.ID()
	}

	var velX, velY float64
	var damage float64

	if isHoming {
		// Homing missile: start with initial velocity towards target
		velX = math.Sin(targetAngle) * homingMissileSpeed
		velY = -math.Cos(targetAngle) * homingMissileSpeed
		damage = homingMissileDamage
	} else {
		// Regular bullet: straight trajectory
		velX = math.Sin(targetAngle) * bulletSpeed
		velY = -math.Cos(targetAngle) * bulletSpeed
		damage = bulletDamage
	}

	// Add ship velocity to bullet (bullets inherit ship momentum)
	velX += ship.vel.x
	velY += ship.vel.y

	bullet := NewBullet(
		turretWorld,
		vec2{x: velX, y: velY},
		ship.faction,
		shipID,
		isHoming,
		targetID,
		damage,
	)

	g.bullets[bullet.ID()] = bullet
}

// updateBullets updates all bullets and removes old ones
func (g *Game) updateBullets(dt float64) {
	// Update bullet positions and homing behavior
	for _, bullet := range g.bullets {
		bullet.age += dt

		// Update homing missiles to track their target
		if bullet.isHoming && bullet.targetID != InvalidEntityID {
			// Try to find target (could be ship or rock)
			var targetPos vec2
			var targetAlive bool

			targetShip := g.GetShip(bullet.targetID)
			if targetShip != nil && targetShip.health > 0 && !g.areAllied(bullet.faction, targetShip.faction) {
				targetPos = targetShip.pos
				targetAlive = true
			} else {
				targetRock := g.GetRock(bullet.targetID)
				if targetRock != nil && targetRock.health > 0 {
					targetPos = targetRock.pos
					targetAlive = true
				}
			}

			if targetAlive {
				// Calculate direction to target
				dx := targetPos.x - bullet.pos.x
				dy := targetPos.y - bullet.pos.y
				dist := math.Hypot(dx, dy)

				if dist > 0.1 {
					// Desired direction to target
					desiredDirX := dx / dist
					desiredDirY := dy / dist

					// Current velocity direction
					speed := math.Hypot(bullet.vel.x, bullet.vel.y)
					if speed < 0.1 {
						speed = homingMissileSpeed
					}
					currentDirX := bullet.vel.x / speed
					currentDirY := bullet.vel.y / speed

					// Calculate angle difference
					desiredAngle := math.Atan2(desiredDirX, -desiredDirY)
					currentAngle := math.Atan2(currentDirX, -currentDirY)
					angleDiff := normalizeAngle(desiredAngle - currentAngle)

					// Turn towards target (limited by turn rate)
					maxTurn := homingMissileTurnRate * dt
					if math.Abs(angleDiff) > maxTurn {
						if angleDiff > 0 {
							angleDiff = maxTurn
						} else {
							angleDiff = -maxTurn
						}
					}

					// Apply rotation to velocity
					newAngle := currentAngle + angleDiff
					bullet.vel.x = math.Sin(newAngle) * homingMissileSpeed
					bullet.vel.y = -math.Cos(newAngle) * homingMissileSpeed
				}
			} else {
				// Target is dead or allied, convert to regular bullet behavior
				bullet.isHoming = false
				bullet.targetID = InvalidEntityID
			}
		}

		bullet.pos.x += bullet.vel.x * dt
		bullet.pos.y += bullet.vel.y * dt
	}

	// Note: Old bullet removal is now handled in removeDeadEntities()
}

// drawBullets renders all bullets
func (g *Game) drawBullets(screen *ebiten.Image, player *Ship) {
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}

	for _, bullet := range g.bullets {
		// Position relative to player
		offsetX := bullet.pos.x - player.pos.x
		offsetY := bullet.pos.y - player.pos.y
		rotated := rotatePoint(vec2{offsetX, offsetY}, -player.angle)
		bulletScreenX := screenCenter.x + rotated.x
		bulletScreenY := screenCenter.y + rotated.y

		// Get bullet color based on faction
		bulletColor := g.colorForFaction(bullet.faction)

		// Draw homing missiles larger and with a different visual style
		if bullet.isHoming {
			// Draw larger circle for homing missile
			drawCircle(screen, bulletScreenX, bulletScreenY, bulletRadius*1.5, bulletColor)
			// Draw inner glow
			drawCircle(screen, bulletScreenX, bulletScreenY, bulletRadius*0.8, color.NRGBA{R: 255, G: 255, B: 255, A: 200})
		} else {
			// Draw regular bullet as a small circle
			drawCircle(screen, bulletScreenX, bulletScreenY, bulletRadius, bulletColor)
		}
	}
}

// findNearestEnemy finds the nearest enemy entity (ship or rock) for a given ship
func (g *Game) findNearestEnemy(ship *Ship) Entity {
	var nearestEnemy Entity
	nearestDist := turretRange + 1.0 // Start beyond range

	// Check ships
	for _, target := range g.ships {
		// Skip self and allies
		if target.ID() == ship.ID() || g.areAllied(ship.faction, target.faction) {
			continue
		}

		// Calculate distance
		dx := target.pos.x - ship.pos.x
		dy := target.pos.y - ship.pos.y
		dist := math.Hypot(dx, dy)

		if dist < nearestDist && dist <= turretRange {
			nearestDist = dist
			nearestEnemy = target
		}
	}

	// Check rocks
	for _, rock := range g.rocks {
		// Calculate distance
		dx := rock.pos.x - ship.pos.x
		dy := rock.pos.y - ship.pos.y
		dist := math.Hypot(dx, dy)

		if dist < nearestDist && dist <= turretRange {
			nearestDist = dist
			nearestEnemy = rock
		}
	}

	return nearestEnemy
}

// shouldFireTurret determines if a turret should fire at a target
func (g *Game) shouldFireTurret(ship *Ship, target Entity, turretIdx int) bool {
	if target == nil {
		return false
	}

	// Check if enough time has passed since last shot
	if g.gameTime-ship.lastFireTime < turretFireRate {
		return false
	}

	// Get turret position in world space
	turretLocal := ship.turretPoints[turretIdx]
	turretWorld := rotatePoint(turretLocal, ship.angle)
	turretWorld.x += ship.pos.x
	turretWorld.y += ship.pos.y

	// Calculate angle to target
	targetPos := target.Position()
	dx := targetPos.x - turretWorld.x
	dy := targetPos.y - turretWorld.y
	targetAngle := math.Atan2(dx, -dy)

	// Calculate angle difference from ship's forward direction
	angleDiff := normalizeAngle(targetAngle - ship.angle)

	// Fire if target is within angle threshold
	return math.Abs(angleDiff) < turretFireAngleThreshold
}

// firePlayerTurrets handles player turret firing when spacebar is pressed
func (g *Game) firePlayerTurrets(player *Ship) {
	// Check if enough time has passed since last shot
	if g.gameTime-player.lastFireTime < turretFireRate {
		return
	}

	// Find nearest enemy to aim at
	target := g.findNearestEnemy(player)

	var targetAngle float64
	var useTarget bool

	if target != nil {
		// Calculate angle to target
		targetPos := target.Position()
		dx := targetPos.x - player.pos.x
		dy := targetPos.y - player.pos.y
		targetAngle = math.Atan2(dx, -dy)
		useTarget = true
	} else {
		// No enemy found, fire forward in direction ship is facing
		targetAngle = player.angle
		useTarget = false
	}

	// Fire turrets - try to fire at target if available, otherwise fire forward
	for i := range player.turretPoints {
		canFire := false

		if useTarget {
			// Check if turret can fire at target (within angle threshold)
			if g.shouldFireTurret(player, target, i) {
				canFire = true
			}
		} else {
			// No target, always fire forward
			canFire = true
		}

		if canFire {
			// Player fires regular bullets (not homing missiles)
			g.spawnBulletWithTarget(player, i, targetAngle, target, false)
			player.lastFireTime = g.gameTime
			break // Only fire one turret per shot to avoid double-firing
		}
	}
}

// updateTurretFiring handles automatic turret firing for NPCs
func (g *Game) updateTurretFiring(ship *Ship, dt float64) {
	// Only NPCs fire automatically (not player)
	if ship.isPlayer {
		return
	}

	// Find nearest enemy
	target := g.findNearestEnemy(ship)
	if target == nil {
		return
	}

	// Calculate angle to target
	targetPos := target.Position()
	dx := targetPos.x - ship.pos.x
	dy := targetPos.y - ship.pos.y
	targetAngle := math.Atan2(dx, -dy)

	// Try to fire each turret
	for i := range ship.turretPoints {
		if g.shouldFireTurret(ship, target, i) {
			// Randomly choose between regular bullet and homing missile (30% chance for homing)
			isHoming := math.Mod(g.gameTime*1000, 100) < 30
			g.spawnBulletWithTarget(ship, i, targetAngle, target, isHoming)
			ship.lastFireTime = g.gameTime
			break // Only fire one turret per frame
		}
	}
}
