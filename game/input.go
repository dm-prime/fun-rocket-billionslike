package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputProvider defines the interface for entity input/behavior
type InputProvider interface {
	// GetMovement returns the desired movement vector (x, y) given the entity's current position
	GetMovement(entityX, entityY float64) (float64, float64)

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
	desiredRotation  float64 // Desired rotation direction (-1 to 1) for auto-targeting
}

// NewPlayerInput creates a new player input provider
func NewPlayerInput() *PlayerInput {
	return &PlayerInput{
		keys:          make([]ebiten.Key, 0, 10),
		MaxTargetRange: 1000.0, // 1000 pixels max range
		HasTarget:     false,
	}
}

// GetMovement returns movement based on arrow keys or WASD
func (p *PlayerInput) GetMovement(entityX, entityY float64) (float64, float64) {
	var moveX, moveY float64
	speed := 200.0 // pixels per second

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		moveX -= speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		moveX += speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		moveY -= speed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		moveY += speed
	}

	return moveX, moveY
}

// GetRotation returns rotation towards target, or manual Q/E keys if no target
// Returns -1 to 1, where 1 is clockwise rotation
func (p *PlayerInput) GetRotation() float64 {
	// If we have a target, rotate towards it automatically
	if p.HasTarget {
		// Return desired rotation direction (-1 to 1)
		// This will be calculated in updatePlayerTargeting
		return p.desiredRotation
	}
	
	// Fallback to manual rotation if no target
	rotationSpeed := 1.0
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		return -rotationSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyE) {
		return rotationSpeed
	}
	return 0
}

// ShouldShoot returns true if there's a target (auto-shoot) or spacebar is pressed
func (p *PlayerInput) ShouldShoot() bool {
	// Auto-shoot when there's a target
	if p.HasTarget {
		return true
	}
	// Fallback to manual shooting
	return ebiten.IsKeyPressed(ebiten.KeySpace)
}

// ShouldRespawn returns true if R key is pressed
func (p *PlayerInput) ShouldRespawn() bool {
	return ebiten.IsKeyPressed(ebiten.KeyR)
}

// Update updates the input state
func (p *PlayerInput) Update(deltaTime float64) {
	// Update pressed keys
	p.keys = inpututil.AppendPressedKeys(p.keys[:0])
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
	PatternTime         float64

	// Enemy type for behavior differentiation
	EnemyType EnemyType
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
		State:        AIStateMoving,
		ShootCooldown: 1.0,
		EnemyType:    EnemyTypeHomingSuicide, // Default
	}
}

// NewAIInputWithType creates a new AI input provider with a specific enemy type
func NewAIInputWithType(enemyType EnemyType) *AIInput {
	shipType := GetShipTypeForEnemyType(enemyType)
	shipConfig := GetShipTypeConfig(shipType)
	ai := &AIInput{
		State:        AIStateMoving,
		ShootCooldown: shipConfig.ShootCooldown,
		EnemyType:    enemyType,
	}
	return ai
}

// GetMovement returns movement towards target
// Note: Speed should be handled by the entity's ship type, but we keep this for compatibility
func (a *AIInput) GetMovement(entityX, entityY float64) (float64, float64) {
	shipType := GetShipTypeForEnemyType(a.EnemyType)
	shipConfig := GetShipTypeConfig(shipType)
	speed := shipConfig.Speed

	// Calculate direction to target
	dx := a.TargetX - entityX
	dy := a.TargetY - entityY

	// Normalize direction
	dist := dx*dx + dy*dy
	if dist > 0 {
		dist = math.Sqrt(dist)
		dx = dx / dist * speed
		dy = dy / dist * speed
	}

	return dx, dy
}

// GetRotation returns rotation towards movement direction
func (a *AIInput) GetRotation() float64 {
	// AI doesn't actively rotate, movement handles direction
	return 0
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

