package game

import (
	"math"
)

// AIBehavior defines different AI behavior patterns
type AIBehavior int

const (
	AIBehaviorStraight AIBehavior = iota
	AIBehaviorCircle
	AIBehaviorChase
	AIBehaviorZigzag
)

// canShipTargetEntity checks if a ship can target a specific entity based on ship config
func canShipTargetEntity(shipType ShipType, target *Entity) bool {
	shipConfig := GetShipTypeConfig(shipType)

	// Check entity type whitelist
	if len(shipConfig.TargetEntityTypes) > 0 {
		found := false
		for _, allowedType := range shipConfig.TargetEntityTypes {
			if target.Type == allowedType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check entity type blacklist
	for _, blockedType := range shipConfig.BlacklistEntityTypes {
		if target.Type == blockedType {
			return false
		}
	}

	// Check ship type whitelist (only for non-projectile entities)
	if target.Type != EntityTypeProjectile && len(shipConfig.TargetShipTypes) > 0 {
		found := false
		for _, allowedShipType := range shipConfig.TargetShipTypes {
			if target.ShipType == allowedShipType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check ship type blacklist (only for non-projectile entities)
	if target.Type != EntityTypeProjectile {
		for _, blockedShipType := range shipConfig.BlacklistShipTypes {
			if target.ShipType == blockedShipType {
				return false
			}
		}
	}

	return true
}

// UpdateAI updates AI input providers with behavior patterns
func UpdateAI(aiInput *AIInput, entity *Entity, player *Entity, world *World, deltaTime float64) {
	if aiInput == nil {
		return
	}

	// Update AI input state
	aiInput.Update(deltaTime)

	// Get entity faction to determine target
	entityFaction := GetEntityFaction(entity)
	targetFaction := GetOppositeFaction(entityFaction)

	// Find nearest target of opposite faction using spatial partitioning
	var targetEntity *Entity
	nearestDistanceSq := math.MaxFloat64

	// Use spatial query to find nearby entities instead of iterating all entities
	searchRadius := 1000.0 // Reasonable search radius
	candidates := world.GetEntitiesInRadius(entity.X, entity.Y, searchRadius)

	for _, candidate := range candidates {
		if !candidate.Active || candidate == entity || candidate.Health <= 0 {
			continue
		}

		// Skip untargetable entities (XP, destroyed indicators, homing rockets, etc.)
		if candidate.Type == EntityTypeXP || candidate.Type == EntityTypeDestroyedIndicator || candidate.Type == EntityTypeHomingRocket {
			continue
		}

		candidateFaction := GetEntityFaction(candidate)
		if candidateFaction == targetFaction {
			// Check if this ship can target this entity based on ship config
			if !canShipTargetEntity(entity.ShipType, candidate) {
				continue
			}
			dx := candidate.X - entity.X
			dy := candidate.Y - entity.Y
			distanceSq := dx*dx + dy*dy // Use squared distance to avoid sqrt

			if distanceSq < nearestDistanceSq {
				nearestDistanceSq = distanceSq
				targetEntity = candidate
			}
		}
	}

	// If no target found in search radius, check player specifically (might be outside radius)
	if targetEntity == nil && player != nil && player.Active {
		playerFaction := GetEntityFaction(player)
		if playerFaction == targetFaction {
			dx := player.X - entity.X
			dy := player.Y - entity.Y
			distanceSq := dx*dx + dy*dy
			if distanceSq < nearestDistanceSq {
				targetEntity = player
			}
		}
	}

	// Update hasTarget flag
	aiInput.hasTarget = targetEntity != nil && targetEntity.Active

	// Calculate target position based on behavior
	targetX := entity.X
	targetY := entity.Y

	// Behavior depends on enemy type
	switch aiInput.EnemyType {
	case EnemyTypeRocket:
		// Direct homing: chase target of opposite faction
		if targetEntity != nil && targetEntity.Active {
			targetX = targetEntity.X
			targetY = targetEntity.Y
		} else {
			// No target, wander
			aiInput.PatternTime += deltaTime
			targetX = entity.X + math.Cos(aiInput.PatternTime)*50
			targetY = entity.Y + math.Sin(aiInput.PatternTime)*50
		}

	case EnemyTypeShooter:
		// Shooter: chase but keep some distance, shoot
		if targetEntity != nil && targetEntity.Active {
			dx := targetEntity.X - entity.X
			dy := targetEntity.Y - entity.Y
			distanceSq := dx*dx + dy*dy
			distance := math.Sqrt(distanceSq)

			if distance > 0 {
				// Try to maintain optimal shooting distance (200-400 pixels)
				optimalDistance := 300.0
				if distance < optimalDistance {
					// Back away slightly
					targetX = entity.X - dx/distance*50
					targetY = entity.Y - dy/distance*50
				} else {
					// Move closer
					targetX = targetEntity.X
					targetY = targetEntity.Y
				}

				// Calculate predictive aim target for shooting
				aimX, aimY, _ := GetAimPoint(entity)
				predictedX, predictedY := CalculatePredictiveAim(aimX, aimY, targetEntity)
				// Store predicted target for rendering
				aiInput.TargetX = predictedX
				aiInput.TargetY = predictedY
			} else {
				// Store movement target
				aiInput.TargetX = targetX
				aiInput.TargetY = targetY
			}
		} else {
			// No target, wander
			aiInput.PatternTime += deltaTime
			targetX = entity.X + math.Cos(aiInput.PatternTime)*50
			targetY = entity.Y + math.Sin(aiInput.PatternTime)*50
			aiInput.TargetX = targetX
			aiInput.TargetY = targetY
		}
	}

	// Update target position (for movement, not shooting)
	// Note: For shooters, TargetX/TargetY is already set to predicted aim position above
	if aiInput.EnemyType != EnemyTypeShooter || player == nil || !player.Active {
		aiInput.TargetX = targetX
		aiInput.TargetY = targetY
	}

	// Calculate desired rotation
	// For shooters, rotate towards predictive aim target (for shooting)
	// For others, rotate towards movement target
	var rotationTargetX, rotationTargetY float64
	if aiInput.EnemyType == EnemyTypeShooter && (targetEntity != nil || (player != nil && player.Active && GetEntityFaction(player) == targetFaction)) {
		// Use predictive aim target for rotation (so ship aims where it will shoot)
		rotationTargetX = aiInput.TargetX
		rotationTargetY = aiInput.TargetY
	} else {
		// Use movement target for rotation
		rotationTargetX = targetX
		rotationTargetY = targetY
	}

	dx := rotationTargetX - entity.X
	dy := rotationTargetY - entity.Y
	distance := math.Sqrt(dx*dx + dy*dy)

	if distance > 1.0 {
		// Calculate angle to target
		// Rotation 0 points right (east), matching rendering convention
		// Atan2(dy, dx) gives angle from positive x-axis
		targetAngle := math.Atan2(dy, dx)

		// Calculate angle difference (normalize to -PI to PI)
		angleDiff := targetAngle - entity.Rotation
		// Normalize angle difference to [-PI, PI]
		for angleDiff > math.Pi {
			angleDiff -= 2 * math.Pi
		}
		for angleDiff < -math.Pi {
			angleDiff += 2 * math.Pi
		}

		// Convert angle difference to rotation input (-1 to 1)
		// Use a dead zone to prevent jittering
		deadZone := 0.1 // radians (~5.7 degrees)
		if math.Abs(angleDiff) > deadZone {
			// Normalize to -1 to 1 range
			maxAngle := math.Pi
			rotationInput := angleDiff / maxAngle
			// Clamp to [-1, 1]
			if rotationInput > 1.0 {
				rotationInput = 1.0
			} else if rotationInput < -1.0 {
				rotationInput = -1.0
			}
			aiInput.DesiredRotation = rotationInput
		} else {
			aiInput.DesiredRotation = 0.0
		}
	} else {
		// Too close, stop rotating
		aiInput.DesiredRotation = 0.0
	}
}

// CreateEnemyAI creates an AI input with a random enemy type
func CreateEnemyAI() *AIInput {
	enemyType := GetRandomEnemyType()
	return NewAIInputWithType(enemyType)
}

// CreateEnemyAIWithType creates an AI input with a specific enemy type
func CreateEnemyAIWithType(enemyType EnemyType) *AIInput {
	return NewAIInputWithType(enemyType)
}

// UpdateEnemyAI updates enemy AI with more sophisticated behaviors
func UpdateEnemyAI(aiInput *AIInput, entity *Entity, player *Entity, deltaTime float64, behavior AIBehavior) {
	if aiInput == nil {
		return
	}

	aiInput.Update(deltaTime)

	switch behavior {
	case AIBehaviorChase:
		if player != nil && player.Active {
			dx := player.X - entity.X
			dy := player.Y - entity.Y
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance > 0 {
				aiInput.TargetX = player.X
				aiInput.TargetY = player.Y
			}
		}

	case AIBehaviorCircle:
		// Circle around a point
		centerX := 5000.0 // World center
		centerY := 5000.0
		radius := 200.0
		angle := aiInput.PatternTime * 0.5
		aiInput.TargetX = centerX + math.Cos(angle)*radius
		aiInput.TargetY = centerY + math.Sin(angle)*radius

	case AIBehaviorZigzag:
		// Zigzag pattern
		aiInput.PatternTime += deltaTime
		zigzagX := math.Sin(aiInput.PatternTime*2.0) * 100
		aiInput.TargetX = entity.X + zigzagX
		aiInput.TargetY = entity.Y - 50 // Move downward

	case AIBehaviorStraight:
		// Move straight down
		aiInput.TargetX = entity.X
		aiInput.TargetY = entity.Y + 100
	}
}
