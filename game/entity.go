package game

import (
	"math"
)

// Entity represents a game entity (player, enemy, or projectile)
type Entity struct {
	// Position in world coordinates
	X, Y float64

	// Velocity in pixels per second
	VX, VY float64

	// Rotation in radians
	Rotation float64

	// Angular velocity in radians per second
	AngularVelocity float64

	// Health points (0 or less means dead)
	Health float64

	// Maximum health
	MaxHealth float64

	// Collision radius in pixels
	Radius float64

	// Input provider for behavior (player input or AI)
	Input InputProvider

	// Entity type identifier
	Type EntityType

	// Ship type (determines stats and graphics)
	ShipType ShipType

	// Faction (determined at spawn time)
	Faction Faction

	// Current cell coordinates (for fast lookup)
	CellX, CellY int

	// Whether this entity is active (used for pooling)
	Active bool

	// Time since creation (for projectiles to avoid immediate collision with shooter)
	Age float64

	// Owner entity (for projectiles, tracks who fired them)
	Owner *Entity

	// NoCollision flag - if true, entity doesn't collide with other entities (except for special cases like explosions)
	NoCollision bool
}

// EntityType identifies the type of entity
type EntityType int

const (
	EntityTypePlayer EntityType = iota
	EntityTypeEnemy
	EntityTypeProjectile
)

// NewEntity creates a new entity with the given parameters
func NewEntity(x, y, radius float64, entityType EntityType, input InputProvider) *Entity {
	// Set default ship type based on entity type
	var shipType ShipType
	switch entityType {
	case EntityTypePlayer:
		shipType = ShipTypePlayer
	case EntityTypeEnemy:
		shipType = ShipTypeHomingSuicide // Default enemy ship type
	default:
		shipType = ShipTypePlayer // Default for projectiles (not really used)
	}
	
	return &Entity{
		X:         x,
		Y:         y,
		Radius:    radius,
		Type:      entityType,
		ShipType:  shipType,
		Input:     input,
		MaxHealth: 100.0,
		Health:    100.0,
		Active:    true,
		Age:       0.0,
		Faction:   FactionEnemy, // Default, should be set explicitly
	}
}

// NewEntityWithShipType creates a new entity with ship type (sets stats from ship type)
// Faction should be set separately after creation
func NewEntityWithShipType(x, y float64, entityType EntityType, shipType ShipType, input InputProvider) *Entity {
	shipConfig := GetShipTypeConfig(shipType)
	entity := &Entity{
		X:         x,
		Y:         y,
		Radius:    shipConfig.Radius,
		Type:      entityType,
		ShipType:  shipType,
		Input:     input,
		MaxHealth: shipConfig.Health,
		Health:    shipConfig.Health,
		Active:    true,
		Age:       0.0,
		Faction:   FactionEnemy, // Default, should be set explicitly
	}
	return entity
}

// Update updates the entity based on input and applies movement
func (e *Entity) Update(deltaTime float64) {
	if !e.Active || e.Health <= 0 {
		return
	}

	// Update age
	e.Age += deltaTime

	if e.Input != nil && e.Type != EntityTypeProjectile {
		// Get ship config for physics properties
		shipConfig := GetShipTypeConfig(e.ShipType)

		// Handle rotation (angular velocity)
		rotationInput := e.Input.GetRotation()
		if math.Abs(rotationInput) > 0.01 {
			// Apply angular acceleration
			e.AngularVelocity += rotationInput * shipConfig.AngularAcceleration * deltaTime
			
			// Clamp to max angular speed
			if e.AngularVelocity > shipConfig.MaxAngularSpeed {
				e.AngularVelocity = shipConfig.MaxAngularSpeed
			} else if e.AngularVelocity < -shipConfig.MaxAngularSpeed {
				e.AngularVelocity = -shipConfig.MaxAngularSpeed
			}
		} else {
			// Apply angular friction
			e.AngularVelocity *= 0.9
		}

		// Update rotation
		e.Rotation += e.AngularVelocity * deltaTime

		// Handle thrust (forward/backward acceleration)
		thrustInput := e.Input.GetThrust()
		if math.Abs(thrustInput) > 0.01 {
			// Calculate forward direction vector
			// Rotation 0 points right (east), matching the rendering convention
			forwardX := math.Cos(e.Rotation)
			forwardY := math.Sin(e.Rotation)

			// Apply acceleration in forward/backward direction
			acceleration := thrustInput * shipConfig.Acceleration * deltaTime
			e.VX += forwardX * acceleration
			e.VY += forwardY * acceleration
		}

		// Apply friction to velocity
		e.VX *= shipConfig.Friction
		e.VY *= shipConfig.Friction

		// Clamp velocity to max speed
		currentSpeed := math.Sqrt(e.VX*e.VX + e.VY*e.VY)
		if currentSpeed > shipConfig.Speed {
			scale := shipConfig.Speed / currentSpeed
			e.VX *= scale
			e.VY *= scale
		}
	} else if e.Type == EntityTypeProjectile {
		// Projectiles maintain their velocity without physics
		// (they're already set when created)
	}

	// Apply velocity to position
	e.X += e.VX * deltaTime
	e.Y += e.VY * deltaTime
}

// DistanceTo calculates the distance to another entity
func (e *Entity) DistanceTo(other *Entity) float64 {
	dx := e.X - other.X
	dy := e.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// IsColliding checks if this entity is colliding with another entity
func (e *Entity) IsColliding(other *Entity) bool {
	distance := e.DistanceTo(other)
	return distance < (e.Radius + other.Radius)
}

// Reset resets the entity for reuse in pooling
func (e *Entity) Reset() {
	e.X = 0
	e.Y = 0
	e.VX = 0
	e.VY = 0
	e.Rotation = 0
	e.AngularVelocity = 0
	e.Health = e.MaxHealth
	e.Active = false
	e.CellX = 0
	e.CellY = 0
	e.Age = 0.0
	e.Faction = FactionEnemy // Reset to default
	e.NoCollision = false
}

