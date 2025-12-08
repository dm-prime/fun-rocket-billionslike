package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputProvider defines the interface for entity input/behavior
type InputProvider interface {
	// GetThrust returns the thrust amount (-1 to 1, where 1 is forward, -1 is backward)
	GetThrust() float64

	// GetRotation returns the desired rotation change (-1 to 1, where 1 is clockwise)
	GetRotation() float64

	// ShouldShoot returns true if the entity should shoot
	ShouldShoot() bool

	// Update updates the input provider state
	Update(deltaTime float64)
}

// PlayerInput provides input from keyboard/gamepad
type PlayerInput struct {
	keys []ebiten.Key

	// Target acquisition AI
	TargetX, TargetY float64
	HasTarget        bool
	MaxTargetRange   float64 // Maximum range to acquire targets

	// Turret rotation (for active turret)
	TurretRotation float64 // Current rotation of the active turret

	// Weapon cooldowns (tracked per weapon type)
	WeaponCooldowns map[WeaponType]float64 // Time since last shot per weapon type
}

// NewPlayerInput creates a new player input provider
func NewPlayerInput() *PlayerInput {
	return &PlayerInput{
		keys:            make([]ebiten.Key, 0, 10),
		MaxTargetRange:  1000.0, // 1000 pixels max range
		HasTarget:       false,
		TurretRotation:  0.0,
		WeaponCooldowns: make(map[WeaponType]float64),
	}
}

// GetThrust returns forward/backward thrust based on W/S or Up/Down keys
// Returns -1 to 1, where 1 is forward thrust, -1 is backward thrust
func (p *PlayerInput) GetThrust() float64 {
	thrust := 0.0
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		thrust += 1.0 // Forward
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		thrust -= 1.0 // Backward
	}
	return thrust
}

// GetRotation returns manual rotation from A/D or Left/Right keys
// Returns -1 to 1, where 1 is clockwise rotation
func (p *PlayerInput) GetRotation() float64 {
	rotation := 0.0
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		rotation -= 1.0 // Counter-clockwise
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		rotation += 1.0 // Clockwise
	}
	return rotation
}

// ShouldShoot returns true if there's a target (auto-shoot) or spacebar is pressed
// Note: Actual firing is controlled by weapon cooldowns in spawnProjectile
func (p *PlayerInput) ShouldShoot() bool {
	// Auto-shoot when there's a target
	if p.HasTarget {
		return true
	}
	// Fallback to manual shooting
	return ebiten.IsKeyPressed(ebiten.KeySpace)
}

// UpdateWeaponCooldown updates cooldown for a specific weapon type
func (p *PlayerInput) UpdateWeaponCooldown(weaponType WeaponType, deltaTime float64) {
	if p.WeaponCooldowns == nil {
		p.WeaponCooldowns = make(map[WeaponType]float64)
	}
	p.WeaponCooldowns[weaponType] += deltaTime
}

// CanShootWeapon checks if a weapon type is ready to fire
func (p *PlayerInput) CanShootWeapon(weaponType WeaponType) bool {
	if p.WeaponCooldowns == nil {
		return true
	}
	// If weapon hasn't been fired yet, it can fire immediately
	timeSinceLastShot, hasBeenFired := p.WeaponCooldowns[weaponType]
	if !hasBeenFired {
		return true
	}
	weaponConfig := GetWeaponConfig(weaponType)
	return timeSinceLastShot >= weaponConfig.Cooldown
}

// ResetWeaponCooldown resets cooldown for a specific weapon type
func (p *PlayerInput) ResetWeaponCooldown(weaponType WeaponType) {
	if p.WeaponCooldowns == nil {
		p.WeaponCooldowns = make(map[WeaponType]float64)
	}
	p.WeaponCooldowns[weaponType] = 0.0
}

// ShouldRespawn returns true if R key is pressed
func (p *PlayerInput) ShouldRespawn() bool {
	return ebiten.IsKeyPressed(ebiten.KeyR)
}

// Update updates the input state
func (p *PlayerInput) Update(deltaTime float64) {
	// Update pressed keys
	p.keys = inpututil.AppendPressedKeys(p.keys[:0])

	// Update weapon cooldowns
	if p.WeaponCooldowns != nil {
		for weaponType := range p.WeaponCooldowns {
			p.WeaponCooldowns[weaponType] += deltaTime
		}
	}
}

// AIInput provides AI-controlled behavior
type AIInput struct {
	// Target position to move towards
	TargetX, TargetY float64

	// Current behavior state
	State AIState

	// Time since last shot
	TimeSinceLastShot float64

	// Shoot cooldown in seconds
	ShootCooldown float64

	// Movement pattern parameters
	PatternX, PatternY float64
	PatternTime        float64

	// Enemy type for behavior differentiation
	EnemyType EnemyType

	// Desired rotation (-1 to 1, where 1 is clockwise)
	DesiredRotation float64
}

// AIState represents the current AI behavior state
type AIState int

const (
	AIStateIdle AIState = iota
	AIStateMoving
	AIStateAttacking
)

// NewAIInput creates a new AI input provider
func NewAIInput() *AIInput {
	return &AIInput{
		State:           AIStateMoving,
		ShootCooldown:   1.0,
		EnemyType:       EnemyTypeRocket, // Default
		DesiredRotation: 0.0,
	}
}

// NewAIInputWithType creates a new AI input provider with a specific enemy type
func NewAIInputWithType(enemyType EnemyType) *AIInput {
	shipType := GetShipTypeForEnemyType(enemyType)
	shipConfig := GetShipTypeConfig(shipType)
	ai := &AIInput{
		State:           AIStateMoving,
		ShootCooldown:   shipConfig.ShootCooldown,
		EnemyType:       enemyType,
		DesiredRotation: 0.0,
	}
	return ai
}

// GetThrust returns forward thrust towards target
// Returns -1 to 1, where 1 is forward thrust, -1 is backward thrust
func (a *AIInput) GetThrust() float64 {
	// AI always tries to move forward (thrust = 1.0)
	// Turning will handle direction changes
	return 1.0
}

// GetRotation returns rotation towards target direction
// Returns -1 to 1, where 1 is clockwise rotation
// This will be calculated based on the angle difference to target
func (a *AIInput) GetRotation() float64 {
	// This will be set by UpdateAI based on target direction
	return a.DesiredRotation
}

// ShouldShoot returns true if AI should shoot
func (a *AIInput) ShouldShoot() bool {
	// Only shooter type enemies shoot
	if a.EnemyType != EnemyTypeShooter {
		return false
	}
	if a.TimeSinceLastShot >= a.ShootCooldown {
		return true
	}
	return false
}

// Update updates the AI state
func (a *AIInput) Update(deltaTime float64) {
	a.TimeSinceLastShot += deltaTime
	a.PatternTime += deltaTime
}
