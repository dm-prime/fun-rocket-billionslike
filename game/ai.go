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

	// Behavior depends on enemy type
	switch aiInput.EnemyType {
	case EnemyTypeHomingSuicide:
		// Direct homing: always chase player directly
		if player != nil && player.Active {
			aiInput.TargetX = player.X
			aiInput.TargetY = player.Y
		} else {
			// No player, wander
			aiInput.PatternTime += deltaTime
			aiInput.TargetX = entity.X + math.Cos(aiInput.PatternTime)*50
			aiInput.TargetY = entity.Y + math.Sin(aiInput.PatternTime)*50
		}

	case EnemyTypeShooter:
		// Shooter: chase but keep some distance, shoot
		if player != nil && player.Active {
			dx := player.X - entity.X
			dy := player.Y - entity.Y
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance > 0 {
				// Try to maintain optimal shooting distance (200-400 pixels)
				optimalDistance := 300.0
				if distance < optimalDistance {
					// Back away slightly
					aiInput.TargetX = entity.X - dx/distance*50
					aiInput.TargetY = entity.Y - dy/distance*50
				} else {
					// Move closer
					aiInput.TargetX = player.X
					aiInput.TargetY = player.Y
				}
			}
		} else {
			// No player, wander
			aiInput.PatternTime += deltaTime
			aiInput.TargetX = entity.X + math.Cos(aiInput.PatternTime)*50
			aiInput.TargetY = entity.Y + math.Sin(aiInput.PatternTime)*50
		}
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

