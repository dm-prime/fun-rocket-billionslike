package game

// WeaponType defines different types of weapons
type WeaponType int

const (
	WeaponTypeBullet WeaponType = iota
	WeaponTypeHomingMissile
	WeaponTypeNone
)

// WeaponConfig holds configuration for each weapon type
type WeaponConfig struct {
	Type            WeaponType
	Damage          float64
	ProjectileSpeed float64
	Cooldown        float64
	Radius          float64 // For projectiles
	InitialVelocity float64 // For homing missiles (launch speed)
	Lifetime        float64 // For homing missiles (time before auto-detonation in seconds)

	// Targeting configuration
	TargetEntityTypes    []EntityType // Whitelist of entity types this weapon can target (empty = all)
	TargetShipTypes      []ShipType   // Whitelist of ship types this weapon can target (empty = all)
	BlacklistEntityTypes []EntityType // Blacklist of entity types this weapon cannot target
	BlacklistShipTypes   []ShipType   // Blacklist of ship types this weapon cannot target
}

// GetWeaponConfig returns configuration for a weapon type
func GetWeaponConfig(weaponType WeaponType) WeaponConfig {
	switch weaponType {
	case WeaponTypeBullet:
		return WeaponConfig{
			Type:                 WeaponTypeBullet,
			Damage:               10.0,
			ProjectileSpeed:      500.0,
			Cooldown:             0.1,
			Radius:               2.5,
			InitialVelocity:      0.0,                                                                            // Not used for bullets
			Lifetime:             0.0,                                                                            // No lifetime limit for bullets
			TargetEntityTypes:    []EntityType{EntityTypeEnemy},                                                  // Only target enemies
			TargetShipTypes:      []ShipType{},                                                                   // All ship types allowed
			BlacklistEntityTypes: []EntityType{EntityTypeProjectile, EntityTypeXP, EntityTypeDestroyedIndicator}, // Don't target projectiles, XP, or indicators
			BlacklistShipTypes:   []ShipType{},                                                                   // No blacklisted ship types
		}
	case WeaponTypeHomingMissile:
		return WeaponConfig{
			Type:                 WeaponTypeHomingMissile,
			Damage:               30.0, // Damage when homing enemy hits
			ProjectileSpeed:      0.0,  // Not used for homing missiles
			Cooldown:             1.0,
			Radius:               0.0,                                                                                                    // Not used for homing missiles
			InitialVelocity:      150.0,                                                                                                  // Launch speed for homing enemy
			Lifetime:             5.0,                                                                                                    // Auto-detonate after 5 seconds
			TargetEntityTypes:    []EntityType{EntityTypeEnemy},                                                                          // Only target enemies
			TargetShipTypes:      []ShipType{ShipTypePlayer, ShipTypeShooter},                                                            // Only target real ships (not rockets)
			BlacklistEntityTypes: []EntityType{EntityTypeProjectile, EntityTypeXP, EntityTypeDestroyedIndicator, EntityTypeHomingRocket}, // Don't target projectiles, XP, indicators, or homing rockets
			BlacklistShipTypes:   []ShipType{},                                                                                           // No blacklisted ship types (using entity type blacklist instead)
		}
	default:
		return GetWeaponConfig(WeaponTypeBullet)
	}
}

// CanShoot checks if a weapon is ready to fire based on time since last shot
// Returns true if the weapon hasn't been fired yet or if enough time has passed
func (wc WeaponConfig) CanShoot(timeSinceLastShot float64, hasBeenFired bool) bool {
	// If weapon hasn't been fired yet, it can fire immediately
	if !hasBeenFired {
		return true
	}
	return timeSinceLastShot >= wc.Cooldown
}
