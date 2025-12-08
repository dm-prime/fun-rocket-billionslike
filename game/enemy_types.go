package game

import "math/rand"

// EnemyType defines different types of enemies
type EnemyType int

const (
	EnemyTypeRocket      EnemyType = iota // Chases player and explodes on contact
	EnemyTypeShooter                      // Shoots rockets at player
	EnemyTypeShooterTwin                  // Shoots rockets and bullets at player

)

// EnemyTypeConfig holds configuration for each enemy type
type EnemyTypeConfig struct {
	Type          EnemyType
	ShipType      ShipType
	Speed         float64
	Health        float64
	Radius        float64
	ShootCooldown float64 // Only used for shooter type
}

// GetEnemyTypeConfig returns configuration for an enemy type
func GetEnemyTypeConfig(enemyType EnemyType) EnemyTypeConfig {
	switch enemyType {
	case EnemyTypeRocket:
		return EnemyTypeConfig{
			Type:     EnemyTypeRocket,
			ShipType: ShipTypeHomingSuicide,
			Speed:    200.0, // Faster than shooter
			Health:   30.0,  // Less health
			Radius:   10.0,
		}
	case EnemyTypeShooter:
		return EnemyTypeConfig{
			Type:          EnemyTypeShooter,
			ShipType:      ShipTypeShooter,
			Speed:         120.0, // Slower
			Health:        50.0,  // More health
			Radius:        12.0,
			ShootCooldown: 1.0 + rand.Float64()*1.5, // 1-2.5 seconds
		}
	case EnemyTypeShooterTwin:
		return EnemyTypeConfig{
			Type:          EnemyTypeShooterTwin,
			ShipType:      ShipTypePlayer,
			Speed:         120.0, // Slower
			Health:        50.0,  // More health
			Radius:        12.0,
			ShootCooldown: 1.0 + rand.Float64()*1.5, // 1-2.5 seconds
		}
	default:
		return GetEnemyTypeConfig(EnemyTypeRocket)
	}
}

// GetRandomEnemyType returns a random enemy type (weighted towards homing suicide)
func GetRandomEnemyType() EnemyType {
	if rand.Float64() < 0.5 {
		return EnemyTypeRocket
	} else if rand.Float64() < 0.8 {
		return EnemyTypeShooter
	} else {
		return EnemyTypeShooterTwin
	}
}
