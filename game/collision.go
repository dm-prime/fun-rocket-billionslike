package game

// CollisionSystem handles collision detection using spatial partitioning
type CollisionSystem struct {
	world *World
}

// NewCollisionSystem creates a new collision system
func NewCollisionSystem(world *World) *CollisionSystem {
	return &CollisionSystem{
		world: world,
	}
}

// CheckCollisions checks for collisions between entities in the world
func (c *CollisionSystem) CheckCollisions() {
	// Track which pairs we've already checked to avoid duplicate checks
	checkedPairs := make(map[*Entity]map[*Entity]bool)

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
				if other == entity || !other.Active || other.Health <= 0 {
					continue
				}

				// Skip if we've already checked this pair
				if checkedPairs[entity] != nil && checkedPairs[entity][other] {
					continue
				}
				if checkedPairs[other] != nil && checkedPairs[other][entity] {
					continue
				}

				// Initialize checked pairs map if needed
				if checkedPairs[entity] == nil {
					checkedPairs[entity] = make(map[*Entity]bool)
				}
				checkedPairs[entity][other] = true

				// Check collision
				if entity.IsColliding(other) {
					c.HandleCollision(entity, other)
				}
			}
		}
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

	// Entity-to-entity collisions (push apart)
	c.PushApart(e1, e2)

	// Apply damage if entities are different types
	if e1.Type != e2.Type {
		e1.Health -= 10.0
		e2.Health -= 10.0
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

	// Ignore collisions for very young projectiles (avoid immediate collision with shooter)
	if projectile.Age < 0.05 { // 50ms grace period
		return
	}

	// Apply damage
	damage := 25.0
	target.Health -= damage

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

