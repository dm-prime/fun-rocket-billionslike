package game

import "math/rand"

// EnemyType defines different types of enemies
type EnemyType int

const (
	EnemyTypeHomingSuicide EnemyType = iota // Chases player and explodes on contact
	EnemyTypeShooter                        // Shoots projectiles at player
)

// EnemyTypeConfig holds configuration for each enemy type
type EnemyTypeConfig struct {
	Type          EnemyType
	Speed         float64
	Health        float64
	Radius        float64
	ShootCooldown float64 // Only used for shooter type
}

// GetEnemyTypeConfig returns configuration for an enemy type
func GetEnemyTypeConfig(enemyType EnemyType) EnemyTypeConfig {
	switch enemyType {
	case EnemyTypeHomingSuicide:
		return EnemyTypeConfig{
			Type:   EnemyTypeHomingSuicide,
			Speed:  200.0, // Faster than shooter
			Health: 30.0,  // Less health
			Radius: 10.0,
		}
	case EnemyTypeShooter:
		return EnemyTypeConfig{
			Type:          EnemyTypeShooter,
			Speed:         120.0, // Slower
			Health:        50.0,  // More health
			Radius:        12.0,
			ShootCooldown: 1.0 + rand.Float64()*1.5, // 1-2.5 seconds
		}
	default:
		return GetEnemyTypeConfig(EnemyTypeHomingSuicide)
	}
}

// GetRandomEnemyType returns a random enemy type (weighted towards homing suicide)
func GetRandomEnemyType() EnemyType {
	// 80% homing suicide, 20% shooter
	if rand.Float64() < 0.8 {
		return EnemyTypeHomingSuicide
	}
	return EnemyTypeShooter
}
