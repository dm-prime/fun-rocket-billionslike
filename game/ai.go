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

// UpdateAI updates AI input providers with behavior patterns
func UpdateAI(aiInput *AIInput, entity *Entity, player *Entity, deltaTime float64) {
	if aiInput == nil {
		return
	}

	// Update AI input state
	aiInput.Update(deltaTime)

	// Get entity faction to determine target
	entityFaction := GetEntityFaction(entity)
	targetFaction := GetOppositeFaction(entityFaction)

	// Find target of opposite faction (use player as primary target for now)
	var targetEntity *Entity
	if player != nil && player.Active {
		playerFaction := GetEntityFaction(player)
		if playerFaction == targetFaction {
			targetEntity = player
		}
	}

	// Calculate target position based on behavior
	targetX := entity.X
	targetY := entity.Y

	// Behavior depends on enemy type
	switch aiInput.EnemyType {
	case EnemyTypeHomingSuicide:
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
			distance := math.Sqrt(dx*dx + dy*dy)

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
		zigzagX := math.Sin(aiInput.PatternTime * 2.0) * 100
		aiInput.TargetX = entity.X + zigzagX
		aiInput.TargetY = entity.Y - 50 // Move downward

	case AIBehaviorStraight:
		// Move straight down
		aiInput.TargetX = entity.X
		aiInput.TargetY = entity.Y + 100
	}
}

