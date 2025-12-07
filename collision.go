package main

import (
	"fmt"
	"math"
)

// CollisionPair represents a collision between two entities
type CollisionPair struct {
	entity1 Entity
	entity2 Entity
	dist    float64
}

// checkCircleCollision checks if two circles are colliding
func checkCircleCollision(pos1 vec2, radius1 float64, pos2 vec2, radius2 float64) (bool, float64) {
	dx := pos1.x - pos2.x
	dy := pos1.y - pos2.y
	dist := math.Hypot(dx, dy)
	collisionRadius := radius1 + radius2
	return dist < collisionRadius, dist
}

// applyCollisionImpulse applies a simple bounce impulse to two entities
func applyCollisionImpulse(pos1, vel1 *vec2, pos2, vel2 *vec2, dt float64) {
	dx := pos2.x - pos1.x
	dy := pos2.y - pos1.y
	dist := math.Hypot(dx, dy)
	
	if dist < distanceThreshold {
		return // Avoid division by zero
	}
	
	// Normalize direction
	dx /= dist
	dy /= dist
	
	// Apply impulse (push entities apart)
	impulse := collisionImpulse * dt
	vel1.x -= dx * impulse
	vel1.y -= dy * impulse
	vel2.x += dx * impulse
	vel2.y += dy * impulse
}

// CollisionSystem manages collision detection and response for all entity types.
// It provides a unified interface for handling collisions between:
// - Ships and ships
// - Ships and rocks
// - Bullets and ships
// - Bullets and rocks
type CollisionSystem struct {
	game *Game
}

// NewCollisionSystem creates a new collision system instance.
// The system operates on the game state to detect and resolve collisions.
func NewCollisionSystem(g *Game) *CollisionSystem {
	return &CollisionSystem{game: g}
}

// CheckAndHandleCollisions checks all collision types and applies appropriate responses.
// This is the main entry point for collision processing each frame.
// It handles damage application, physics responses, and cleanup in a unified way.
func (cs *CollisionSystem) CheckAndHandleCollisions(dt float64) {
	cs.checkShipToShipCollisions(dt)
	cs.checkShipToRockCollisions(dt)
	cs.checkBulletCollisions(dt)
}

// checkShipToShipCollisions handles ship-ship collisions
func (cs *CollisionSystem) checkShipToShipCollisions(dt float64) {
	g := cs.game
	checked := make(map[string]bool)
	
	for id1, ship1 := range g.ships {
		if ship1.health <= 0 {
			continue
		}
		
		for id2, ship2 := range g.ships {
			if id1 >= id2 || ship2.health <= 0 {
				continue
			}
			
			// Create unique key
			key := fmt.Sprintf("%d-%d", id1, id2)
			if checked[key] {
				continue
			}
			checked[key] = true
			
			// Check collision
			colliding, _ := checkCircleCollision(
				ship1.pos, ship1.CollisionRadius(),
				ship2.pos, ship2.CollisionRadius(),
			)
			
			if colliding {
				// Apply damage
				damage := shipCollisionDamage
				ship1.OnCollision(ship2, damage)
				ship2.OnCollision(ship1, damage)
				
				// Apply impulse
				applyCollisionImpulse(&ship1.pos, &ship1.vel, &ship2.pos, &ship2.vel, dt)
			}
		}
	}
}

// checkShipToRockCollisions handles ship-rock collisions
func (cs *CollisionSystem) checkShipToRockCollisions(dt float64) {
	g := cs.game
	
	for _, ship := range g.ships {
		if ship.health <= 0 {
			continue
		}
		
		for _, rock := range g.rocks {
			if rock.health <= 0 {
				continue
			}
			
			// Check collision
			colliding, _ := checkCircleCollision(
				ship.pos, ship.CollisionRadius(),
				rock.pos, rock.CollisionRadius(),
			)
			
			if colliding {
				// Apply damage
				damage := rockCollisionDamage
				ship.OnCollision(rock, damage)
				rock.OnCollision(ship, damage)
				
				// Apply impulse
				applyCollisionImpulse(&ship.pos, &ship.vel, &rock.pos, &rock.vel, dt)
			}
		}
	}
}

// checkBulletCollisions handles bullet collisions with ships and rocks
func (cs *CollisionSystem) checkBulletCollisions(dt float64) {
	g := cs.game
	bulletsToRemove := make([]EntityID, 0)
	
	for bulletID, bullet := range g.bullets {
		hitSomething := false
		
		// Check ship collisions
		for shipID, ship := range g.ships {
			if ship.health <= 0 {
				continue
			}
			
			// Skip if bullet was fired by this ship
			if bullet.ownerID == shipID {
				continue
			}
			
			// Skip if allied
			if g.areAllied(bullet.faction, ship.faction) {
				continue
			}
			
			// Check collision
			colliding, _ := checkCircleCollision(
				bullet.pos, bulletRadius,
				ship.pos, ship.CollisionRadius(),
			)
			
			if colliding {
				ship.OnCollision(bullet, bullet.damage)
				hitSomething = true
				break
			}
		}
		
		if hitSomething {
			bulletsToRemove = append(bulletsToRemove, bulletID)
			continue
		}
		
		// Check rock collisions
		for _, rock := range g.rocks {
			if rock.health <= 0 {
				continue
			}
			
			// Check collision
			colliding, _ := checkCircleCollision(
				bullet.pos, bulletRadius,
				rock.pos, rock.CollisionRadius(),
			)
			
			if colliding {
				rock.OnCollision(bullet, bullet.damage)
				hitSomething = true
				break
			}
		}
		
		if hitSomething {
			bulletsToRemove = append(bulletsToRemove, bulletID)
		}
	}
	
	// Remove bullets that hit something
	for _, bulletID := range bulletsToRemove {
		g.RemoveBullet(bulletID)
	}
}

