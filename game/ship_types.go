package game

import (
	"image/color"
	"math/rand"
)

// ShipType defines different types of ships
type ShipType int

const (
	ShipTypePlayer ShipType = iota
	ShipTypeHomingSuicide
	ShipTypeShooter
	ShipTypeCount // Total number of ship types
)

// ShipTypeConfig holds configuration for each ship type
type ShipTypeConfig struct {
	Type         ShipType
	Name         string
	Speed        float64
	Health       float64
	Radius       float64
	ShootCooldown float64 // Only used for ships that can shoot
	Color        color.RGBA
	Shape        ShipShape
}

// ShipShape defines the visual shape of a ship
type ShipShape int

const (
	ShipShapeCircle ShipShape = iota
	ShipShapeTriangle
	ShipShapeSquare
	ShipShapeDiamond
)

// GetShipTypeConfig returns configuration for a ship type
func GetShipTypeConfig(shipType ShipType) ShipTypeConfig {
	switch shipType {
	case ShipTypePlayer:
		return ShipTypeConfig{
			Type:          ShipTypePlayer,
			Name:          "Player",
			Speed:         200.0,
			Health:        100.0,
			Radius:        15.0,
			ShootCooldown: 0.1, // Very fast shooting
			Color:         color.RGBA{0, 255, 0, 255}, // Green
			Shape:         ShipShapeTriangle,
		}
	case ShipTypeHomingSuicide:
		return ShipTypeConfig{
			Type:          ShipTypeHomingSuicide,
			Name:          "Homing Suicide",
			Speed:         200.0,
			Health:        30.0,
			Radius:        10.0,
			ShootCooldown: 0.0, // Doesn't shoot
			Color:         color.RGBA{255, 100, 0, 255}, // Orange
			Shape:         ShipShapeTriangle,
		}
	case ShipTypeShooter:
		return ShipTypeConfig{
			Type:          ShipTypeShooter,
			Name:          "Shooter",
			Speed:         120.0,
			Health:        50.0,
			Radius:        12.0,
			ShootCooldown: 1.0 + rand.Float64()*1.5, // 1-2.5 seconds
			Color:         color.RGBA{255, 0, 0, 255}, // Red
			Shape:         ShipShapeTriangle,
		}
	default:
		return GetShipTypeConfig(ShipTypePlayer)
	}
}

// GetShipTypeForEnemyType returns the ship type for a given enemy type
func GetShipTypeForEnemyType(enemyType EnemyType) ShipType {
	switch enemyType {
	case EnemyTypeHomingSuicide:
		return ShipTypeHomingSuicide
	case EnemyTypeShooter:
		return ShipTypeShooter
	default:
		return ShipTypeHomingSuicide
	}
}

// GetEnemyTypeForShipType returns the enemy type for a given ship type
func GetEnemyTypeForShipType(shipType ShipType) EnemyType {
	switch shipType {
	case ShipTypeHomingSuicide:
		return EnemyTypeHomingSuicide
	case ShipTypeShooter:
		return EnemyTypeShooter
	default:
		return EnemyTypeHomingSuicide
	}
}

