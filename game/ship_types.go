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

// TurretMountPoint defines a turret mount position on a ship
type TurretMountPoint struct {
	OffsetX      float64 // X offset from ship center (relative to ship forward direction)
	OffsetY      float64 // Y offset from ship center (relative to ship forward direction)
	Angle        float64 // Angle offset from ship forward direction (in radians)
	Active       bool    // Whether this mount point has an active turret
	BarrelLength float64 // Length of the barrel (where bullets spawn)
}

// ShipTypeConfig holds configuration for each ship type
type ShipTypeConfig struct {
	Type          ShipType
	Name          string
	Speed         float64
	Health        float64
	Radius        float64
	ShootCooldown float64 // Only used for ships that can shoot
	Color         color.RGBA
	Shape         ShipShape
	TurretMounts  []TurretMountPoint // Turret mount points on this ship
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
			Radius:        10.0,                       // Smaller collision radius
			ShootCooldown: 0.1,                        // Very fast shooting
			Color:         color.RGBA{0, 255, 0, 255}, // Green
			Shape:         ShipShapeTriangle,
			TurretMounts: []TurretMountPoint{
				{OffsetX: 0.0, OffsetY: -8.0, Angle: 0.0, Active: true, BarrelLength: 12.0}, // Front mount (active)
				{OffsetX: 0.0, OffsetY: 5.0, Angle: 0.0, Active: false, BarrelLength: 12.0}, // Rear mount (inactive)
			},
		}
	case ShipTypeHomingSuicide:
		return ShipTypeConfig{
			Type:          ShipTypeHomingSuicide,
			Name:          "Homing Suicide",
			Speed:         200.0,
			Health:        30.0,
			Radius:        10.0,
			ShootCooldown: 0.0,                          // Doesn't shoot
			Color:         color.RGBA{255, 100, 0, 255}, // Orange
			Shape:         ShipShapeTriangle,
			TurretMounts:  []TurretMountPoint{}, // No turrets
		}
	case ShipTypeShooter:
		return ShipTypeConfig{
			Type:          ShipTypeShooter,
			Name:          "Shooter",
			Speed:         120.0,
			Health:        50.0,
			Radius:        12.0,
			ShootCooldown: 1.0 + rand.Float64()*1.5,   // 1-2.5 seconds
			Color:         color.RGBA{255, 0, 0, 255}, // Red
			Shape:         ShipShapeTriangle,
			TurretMounts:  []TurretMountPoint{}, // No turrets (shoots from center)
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
