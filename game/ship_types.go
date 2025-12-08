package game

import (
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
	OffsetX      float64    // X offset from ship center (relative to ship forward direction)
	OffsetY      float64    // Y offset from ship center (relative to ship forward direction)
	Angle        float64    // Angle offset from ship forward direction (in radians)
	Active       bool       // Whether this mount point has an active turret
	BarrelLength float64    // Length of the barrel (where bullets spawn)
	WeaponType   WeaponType // Type of weapon mounted on this turret
}

// ShipTypeConfig holds configuration for each ship type
type ShipTypeConfig struct {
	Type          ShipType
	Name          string
	Speed         float64 // Max speed (pixels per second)
	Acceleration  float64 // Thrust acceleration (pixels per second squared)
	Health        float64
	Radius        float64
	ShootCooldown float64 // Only used for ships that can shoot
	Shape         ShipShape
	TurretMounts  []TurretMountPoint // Turret mount points on this ship
	// Physics properties
	AngularAcceleration float64 // Angular acceleration (radians per second squared)
	MaxAngularSpeed     float64 // Max angular velocity (radians per second)
	Friction            float64 // Velocity damping factor (0-1, higher = less friction)
	// Weapon properties
	DefaultWeaponType WeaponType // Default weapon type for ships without turrets
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
			Type:                ShipTypePlayer,
			Name:                "Player",
			Speed:               200.0, // Max speed
			Acceleration:        400.0, // Thrust acceleration
			Health:              100.0,
			Radius:              10.0, // Smaller collision radius
			ShootCooldown:       0.1,  // Very fast shooting
			Shape:               ShipShapeTriangle,
			AngularAcceleration: 5.0,              // Radians per second squared
			MaxAngularSpeed:     3.0,              // Radians per second
			Friction:            0.98,             // Slight friction
			DefaultWeaponType:   WeaponTypeBullet, // Fallback weapon type
			TurretMounts: []TurretMountPoint{
				{OffsetX: 0.0, OffsetY: -8.0, Angle: 0.0, Active: true, BarrelLength: 12.0, WeaponType: WeaponTypeBullet},        // Right mount (active) - bullets
				{OffsetX: 16.0, OffsetY: 0.0, Angle: 0.0, Active: true, BarrelLength: 10.0, WeaponType: WeaponTypeHomingMissile}, // Front mount (active) - rockets
				{OffsetX: 0.0, OffsetY: 8.0, Angle: 0.0, Active: true, BarrelLength: 12.0, WeaponType: WeaponTypeBullet},         // Left mount (active) - bullets

			},
		}
	case ShipTypeHomingSuicide:
		return ShipTypeConfig{
			Type:                ShipTypeHomingSuicide,
			Name:                "Homing Rocket",
			Speed:               200.0, // Max speed
			Acceleration:        350.0, // Thrust acceleration
			Health:              1.0,
			Radius:              6.0,
			ShootCooldown:       0.0, // Doesn't shoot
			Shape:               ShipShapeTriangle,
			AngularAcceleration: 4.0,                  // Radians per second squared
			MaxAngularSpeed:     2.5,                  // Radians per second
			Friction:            0.97,                 // Moderate friction
			DefaultWeaponType:   WeaponTypeNone,       // Not used (doesn't shoot)
			TurretMounts:        []TurretMountPoint{}, // No turrets
		}
	case ShipTypeShooter:
		return ShipTypeConfig{
			Type:                ShipTypeShooter,
			Name:                "Shooter",
			Speed:               120.0, // Max speed
			Acceleration:        250.0, // Thrust acceleration
			Health:              50.0,
			Radius:              12.0,
			ShootCooldown:       1.0 + rand.Float64()*1.5, // 1-2.5 seconds
			Shape:               ShipShapeTriangle,
			AngularAcceleration: 3.0,                     // Radians per second squared
			MaxAngularSpeed:     2.0,                     // Radians per second
			Friction:            0.96,                    // More friction
			DefaultWeaponType:   WeaponTypeHomingMissile, // Spawns homing enemies
			TurretMounts: []TurretMountPoint{
				{OffsetX: 0.0, OffsetY: 0.0, Angle: 0.0, Active: true, BarrelLength: 12.0, WeaponType: WeaponTypeHomingMissile},
			}, // No turrets (shoots from center)
		}
	default:
		return GetShipTypeConfig(ShipTypePlayer)
	}
}
