package game

import "math"

// AIContext is passed to AI scripts as input
// Contains information about the entity and its surroundings
type AIContext struct {
	// Entity state
	EntityX        float64 `json:"entityX"`
	EntityY        float64 `json:"entityY"`
	EntityVX       float64 `json:"entityVX"`
	EntityVY       float64 `json:"entityVY"`
	EntityRotation float64 `json:"entityRotation"`
	EntityHealth   float64 `json:"entityHealth"`
	EntityMaxHP    float64 `json:"entityMaxHP"`

	// Player state (target)
	PlayerX        float64 `json:"playerX"`
	PlayerY        float64 `json:"playerY"`
	PlayerVX       float64 `json:"playerVX"`
	PlayerVY       float64 `json:"playerVY"`
	PlayerRotation float64 `json:"playerRotation"`
	PlayerHealth   float64 `json:"playerHealth"`
	PlayerActive   bool    `json:"playerActive"`

	// Computed values
	DistanceToPlayer float64 `json:"distanceToPlayer"`
	AngleToPlayer    float64 `json:"angleToPlayer"`

	// Nearby entities
	NearbyEnemies []EntityInfo `json:"nearbyEnemies"`
	NearbyAllies  []EntityInfo `json:"nearbyAllies"`

	// Game state
	DeltaTime float64 `json:"deltaTime"`
	GameTime  float64 `json:"gameTime"`
}

// EntityInfo provides information about a nearby entity
type EntityInfo struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	VX         float64 `json:"vx"`
	VY         float64 `json:"vy"`
	Rotation   float64 `json:"rotation"`
	Health     float64 `json:"health"`
	MaxHealth  float64 `json:"maxHealth"`
	Distance   float64 `json:"distance"`
	Angle      float64 `json:"angle"`
	EntityType string  `json:"entityType"`
	ShipType   string  `json:"shipType"`
}

// AIDecision is returned from AI scripts
// Contains the entity's desired actions
type AIDecision struct {
	// Movement direction (-1 to 1 for each axis)
	MoveX float64 `json:"moveX"`
	MoveY float64 `json:"moveY"`

	// Rotation control
	// If TargetAngle is set, entity rotates towards that angle
	// Otherwise, uses RotationSpeed (-1 to 1, where 1 is clockwise)
	TargetAngle   *float64 `json:"targetAngle,omitempty"`
	RotationSpeed float64  `json:"rotationSpeed"`

	// Combat
	ShouldShoot bool `json:"shouldShoot"`

	// Thrust control (-1 to 1, where 1 is forward)
	Thrust float64 `json:"thrust"`
}

// BuildAIContext creates an AIContext from an entity and game state
func BuildAIContext(entity *Entity, player *Entity, world *World, deltaTime, gameTime float64) AIContext {
	ctx := AIContext{
		EntityX:        entity.X,
		EntityY:        entity.Y,
		EntityVX:       entity.VX,
		EntityVY:       entity.VY,
		EntityRotation: entity.Rotation,
		EntityHealth:   entity.Health,
		EntityMaxHP:    entity.MaxHealth,
		DeltaTime:      deltaTime,
		GameTime:       gameTime,
	}

	// Set player info if available
	if player != nil && player.Active {
		ctx.PlayerX = player.X
		ctx.PlayerY = player.Y
		ctx.PlayerVX = player.VX
		ctx.PlayerVY = player.VY
		ctx.PlayerRotation = player.Rotation
		ctx.PlayerHealth = player.Health
		ctx.PlayerActive = true
		ctx.DistanceToPlayer = entity.DistanceTo(player)
		ctx.AngleToPlayer = angleTo(entity, player)
	}

	// Find nearby entities
	if world != nil {
		nearbyRadius := 500.0
		nearby := world.GetEntitiesInRadius(entity.X, entity.Y, nearbyRadius)

		entityFaction := GetEntityFaction(entity)

		for _, other := range nearby {
			if other == entity || !other.Active {
				continue
			}
			// Skip projectiles and XP
			if other.Type == EntityTypeProjectile || other.Type == EntityTypeXP || other.Type == EntityTypeDestroyedIndicator {
				continue
			}

			info := EntityInfo{
				X:          other.X,
				Y:          other.Y,
				VX:         other.VX,
				VY:         other.VY,
				Rotation:   other.Rotation,
				Health:     other.Health,
				MaxHealth:  other.MaxHealth,
				Distance:   entity.DistanceTo(other),
				Angle:      angleTo(entity, other),
				EntityType: entityTypeToString(other.Type),
				ShipType:   shipTypeToString(other.ShipType),
			}

			otherFaction := GetEntityFaction(other)
			if otherFaction == entityFaction {
				ctx.NearbyAllies = append(ctx.NearbyAllies, info)
			} else {
				ctx.NearbyEnemies = append(ctx.NearbyEnemies, info)
			}
		}
	}

	return ctx
}

// entityTypeToString converts EntityType to string for JSON
func entityTypeToString(et EntityType) string {
	switch et {
	case EntityTypePlayer:
		return "player"
	case EntityTypeEnemy:
		return "enemy"
	case EntityTypeProjectile:
		return "projectile"
	case EntityTypeXP:
		return "xp"
	case EntityTypeDestroyedIndicator:
		return "destroyed"
	default:
		return "unknown"
	}
}

// shipTypeToString converts ShipType to string for JSON
func shipTypeToString(st ShipType) string {
	switch st {
	case ShipTypePlayer:
		return "player"
	case ShipTypeHomingSuicide:
		return "homingSuicide"
	case ShipTypeShooter:
		return "shooter"
	default:
		return "unknown"
	}
}

// angleTo calculates the angle from one entity to another
func angleTo(from, to *Entity) float64 {
	dx := to.X - from.X
	dy := to.Y - from.Y
	return angleFromDeltas(dx, dy)
}

// angleFromDeltas calculates angle from delta x and delta y
func angleFromDeltas(dx, dy float64) float64 {
	return atan2(dy, dx)
}

// atan2 is a simple wrapper for math.Atan2
func atan2(y, x float64) float64 {
	return math.Atan2(y, x)
}
