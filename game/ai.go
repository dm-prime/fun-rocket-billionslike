package game

import (
	"math"
	"math/rand"
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

	// Simple chase behavior: move towards player
	if player != nil && player.Active {
		dx := player.X - entity.X
		dy := player.Y - entity.Y
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance > 0 {
			// Normalize and set target
			aiInput.TargetX = entity.X + dx/distance*100
			aiInput.TargetY = entity.Y + dy/distance*100
		}
	} else {
		// No player, use pattern movement
		aiInput.PatternTime += deltaTime
		aiInput.TargetX = entity.X + math.Cos(aiInput.PatternTime)*50
		aiInput.TargetY = entity.Y + math.Sin(aiInput.PatternTime)*50
	}
}

// CreateEnemyAI creates an AI input with a random behavior pattern
func CreateEnemyAI() *AIInput {
	ai := NewAIInput()
	ai.ShootCooldown = 0.5 + rand.Float64()*1.5 // Random cooldown between 0.5-2 seconds
	return ai
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

