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
func (g *Game) spawnBulletWithTarget(ship *Ship, turretIdx int, targetAngle float64, target *Ship, isHoming bool) {
	if turretIdx < 0 || turretIdx >= len(ship.turretPoints) {
		return
	}

	// Get turret position in world space
	turretLocal := ship.turretPoints[turretIdx]
	turretWorld := rotatePoint(turretLocal, ship.angle)
	turretWorld.x += ship.pos.x
	turretWorld.y += ship.pos.y

	// Find ship index
	shipIdx := -1
	for i := range g.ships {
		if &g.ships[i] == ship {
			shipIdx = i
			break
		}
	}

	// Find target index if homing
	targetIdx := -1
	if isHoming && target != nil {
		for i := range g.ships {
			if &g.ships[i] == target {
				targetIdx = i
				break
			}
		}
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

	bullet := Bullet{
		pos:       turretWorld,
		vel:       vec2{x: velX, y: velY},
		age:       0,
		faction:   ship.faction,
		shipIdx:   shipIdx,
		isHoming:  isHoming,
		targetIdx: targetIdx,
		damage:    damage,
	}

	g.bullets = append(g.bullets, bullet)
}

// updateBullets updates all bullets and removes old ones
func (g *Game) updateBullets(dt float64) {
	// Update bullet positions and homing behavior
	for i := range g.bullets {
		bullet := &g.bullets[i]
		bullet.age += dt

		// Update homing missiles to track their target
		if bullet.isHoming && bullet.targetIdx >= 0 && bullet.targetIdx < len(g.ships) {
			target := &g.ships[bullet.targetIdx]
			
			// Check if target is still valid (not dead, not allied)
			if target.health > 0 && !g.areAllied(bullet.faction, target.faction) {
				// Calculate direction to target
				dx := target.pos.x - bullet.pos.x
				dy := target.pos.y - bullet.pos.y
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
			}
		}

		bullet.pos.x += bullet.vel.x * dt
		bullet.pos.y += bullet.vel.y * dt
	}

	// Remove old bullets
	validBullets := g.bullets[:0]
	for i := range g.bullets {
		bullet := &g.bullets[i]
		lifetime := bulletLifetime
		if bullet.isHoming {
			lifetime = homingMissileLifetime
		}
		if bullet.age < lifetime {
			validBullets = append(validBullets, *bullet)
		}
	}
	g.bullets = validBullets
}

// drawBullets renders all bullets
func (g *Game) drawBullets(screen *ebiten.Image, player *Ship) {
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}

	for i := range g.bullets {
		bullet := &g.bullets[i]
		
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

// findNearestEnemy finds the nearest enemy ship for a given ship
func (g *Game) findNearestEnemy(ship *Ship) *Ship {
	var nearestEnemy *Ship
	nearestDist := turretRange + 1.0 // Start beyond range

	for i := range g.ships {
		target := &g.ships[i]
		
		// Skip self, rocks, and allies
		if target == ship || g.isRock(target) || g.areAllied(ship.faction, target.faction) {
			continue
		}

		// Calculate distance
		dx := target.pos.x - ship.pos.x
		dy := target.pos.y - ship.pos.y
		dist := math.Hypot(dx, dy)

		if dist < nearestDist {
			nearestDist = dist
			nearestEnemy = target
		}
	}

	return nearestEnemy
}

// shouldFireTurret determines if a turret should fire at a target
func (g *Game) shouldFireTurret(ship *Ship, target *Ship, turretIdx int) bool {
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
	dx := target.pos.x - turretWorld.x
	dy := target.pos.y - turretWorld.y
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
		dx := target.pos.x - player.pos.x
		dy := target.pos.y - player.pos.y
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
			var targetShip *Ship = nil
			if useTarget {
				targetShip = target
			}
			g.spawnBulletWithTarget(player, i, targetAngle, targetShip, false)
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

	// Skip rocks
	if g.isRock(ship) {
		return
	}

	// Find nearest enemy
	target := g.findNearestEnemy(ship)
	if target == nil {
		return
	}

	// Calculate angle to target
	dx := target.pos.x - ship.pos.x
	dy := target.pos.y - ship.pos.y
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

