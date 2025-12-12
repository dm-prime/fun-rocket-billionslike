package game

// CollisionSystem handles collision detection using spatial partitioning
type CollisionSystem struct {
	world *World
	game  *Game // Reference to game for creating destroyed indicators
}

// NewCollisionSystem creates a new collision system
func NewCollisionSystem(world *World) *CollisionSystem {
	return &CollisionSystem{
		world: world,
		game:  nil, // Will be set by SetGame
	}
}

// SetGame sets the game reference for creating destroyed indicators
func (c *CollisionSystem) SetGame(game *Game) {
	c.game = game
}

// CheckCollisions checks for collisions between entities in the world
func (c *CollisionSystem) CheckCollisions() {
	// Track checked pairs using a simple approach: mark entities as processed
	// Since we iterate through AllEntities in order, we can use indices
	processed := make(map[*Entity]bool, len(c.world.AllEntities))

	// Iterate through all entities
	for _, entity := range c.world.AllEntities {
		if !entity.Active || entity.Health <= 0 {
			continue
		}

		// Get cells that this entity overlaps with
		cells := c.world.GetCellsForEntity(entity)

		// Check collisions with entities in these cells
		for _, cell := range cells {
			for _, other := range cell.GetActiveEntities() {
				// Skip self, inactive, or dead entities
				if other == entity || !other.Active || other.Health <= 0 {
					continue
				}

				// Only check pairs where the other entity hasn't been processed yet
				// This ensures each pair is checked exactly once
				if processed[other] {
					continue
				}

				// Skip collision if both entities have NoCollision flag (they pass through each other)
				if entity.NoCollision && other.NoCollision {
					continue
				}

				// Skip collision if one has NoCollision and they're the same faction (homing rockets pass through allies)
				if entity.NoCollision || other.NoCollision {
					if GetEntityFaction(entity) == GetEntityFaction(other) {
						continue
					}
					// Different factions - allow collision check (for homing rocket explosions)
				}

				// Check collision
				if entity.IsColliding(other) {
					c.HandleCollision(entity, other)
				}
			}
		}
		
		// Mark this entity as processed
		processed[entity] = true
	}
}

// HandleCollision handles a collision between two entities
func (c *CollisionSystem) HandleCollision(e1, e2 *Entity) {
	// Projectile collisions
	if e1.Type == EntityTypeProjectile {
		c.HandleProjectileCollision(e1, e2)
		return
	}
	if e2.Type == EntityTypeProjectile {
		c.HandleProjectileCollision(e2, e1)
		return
	}

	// Check if either entity is a homing suicide enemy colliding with opposite faction
	// Homing suicide enemies explode on contact with opposite faction (even if NoCollision is set)
	if e1.ShipType == ShipTypeHomingSuicide && e2.ShipType != ShipTypeHomingSuicide {
		if GetEntityFaction(e1) != GetEntityFaction(e2) {
			// Different factions - homing suicide explodes
			e2.Health -= 50.0 // Damage target
			e1.Active = false // Destroy homing enemy
			e1.Health = 0
			return
		}
		// Same faction - skip collision if NoCollision is set
		if e1.NoCollision {
			return
		}
	}
	if e2.ShipType == ShipTypeHomingSuicide && e1.ShipType != ShipTypeHomingSuicide {
		if GetEntityFaction(e1) != GetEntityFaction(e2) {
			// Different factions - homing suicide explodes
			e1.Health -= 50.0 // Damage target
			e2.Active = false // Destroy homing enemy
			e2.Health = 0
			return
		}
		// Same faction - skip collision if NoCollision is set
		if e2.NoCollision {
			return
		}
	}

	// Skip push apart if either entity has NoCollision flag
	if e1.NoCollision || e2.NoCollision {
		return
	}

	// Entity-to-entity collisions (push apart)
	c.PushApart(e1, e2)

	// Apply damage if entities are different types (but not suicide enemies, handled above)
	if e1.Type != e2.Type {
		// Only apply collision damage if not a suicide enemy
		isSuicide1 := false
		isSuicide2 := false
		if aiInput1, ok := e1.Input.(*AIInput); ok {
			isSuicide1 = aiInput1.EnemyType == EnemyTypeRocket
		}
		if aiInput2, ok := e2.Input.(*AIInput); ok {
			isSuicide2 = aiInput2.EnemyType == EnemyTypeRocket
		}

		if !isSuicide1 && !isSuicide2 {
			e1.Health -= 10.0
			e2.Health -= 10.0
		}
	}
}

// HandleProjectileCollision handles collision between a projectile and an entity
func (c *CollisionSystem) HandleProjectileCollision(projectile, target *Entity) {
	// Don't hit same type
	if projectile.Type == target.Type {
		return
	}

	// Don't hit projectiles with projectiles
	if target.Type == EntityTypeProjectile {
		return
	}

	// Don't hit the owner of the projectile (player bullets can't hit player)
	if projectile.Owner != nil && projectile.Owner == target {
		return
	}

	// Ignore collisions for very young projectiles (avoid immediate collision with shooter)
	if projectile.Age < 0.05 { // 50ms grace period
		return
	}

	// Apply damage
	damage := 25.0
	oldHealth := target.Health
	target.Health -= damage

	// Check if enemy was destroyed by player projectile
	if target.Type == EntityTypeEnemy && oldHealth > 0 && target.Health <= 0 {
		// Enemy was just killed - check if projectile is from player faction
		if projectile.Owner != nil && projectile.Owner.Faction == FactionPlayer {
			// Create destroyed indicator in yellow (bullet color)
			if c.game != nil {
				c.game.createDestroyedIndicatorYellow(target.X, target.Y)
			}
		}
	}

	// Deactivate projectile
	projectile.Active = false
	projectile.Health = 0
}

// PushApart pushes two entities apart to resolve collision
func (c *CollisionSystem) PushApart(e1, e2 *Entity) {
	// Calculate direction from e1 to e2
	dx := e2.X - e1.X
	dy := e2.Y - e1.Y
	distance := e1.DistanceTo(e2)

	if distance == 0 {
		// Entities are exactly on top of each other, separate randomly
		dx = 1.0
		dy = 1.0
		distance = 1.414
	}

	// Normalize direction
	dx /= distance
	dy /= distance

	// Calculate overlap
	overlap := (e1.Radius + e2.Radius) - distance

	if overlap > 0 {
		// Push apart by half the overlap each
		separation := overlap * 0.5
		e1.X -= dx * separation
		e1.Y -= dy * separation
		e2.X += dx * separation
		e2.Y += dy * separation

		// Update cell membership
		c.world.UpdateEntityCell(e1)
		c.world.UpdateEntityCell(e2)
	}
}

// MoveEntity updates entity position and cell membership
func (c *CollisionSystem) MoveEntity(entity *Entity) {
	if !entity.Active {
		return
	}

	// Update cell membership if entity moved
	c.world.UpdateEntityCell(entity)
}
