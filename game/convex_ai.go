package game

import (
	"fmt"
	"math"
)

// ConvexAIInput provides AI behavior driven by JavaScript scripts
// fetched from Convex database
type ConvexAIInput struct {
	// Script code to execute
	scriptCode string

	// WASM runner for script execution
	runner *WASMRunner

	// Reference to the entity (set during UpdateAI)
	entity *Entity

	// Reference to the player (set during UpdateAI)
	player *Entity

	// Reference to the world (set during UpdateAI)
	world *World

	// Cached decision from last script execution
	lastDecision AIDecision

	// Time tracking
	gameTime  float64
	deltaTime float64

	// Script execution frequency (to avoid running every frame)
	scriptCooldown float64
	timeSinceExec  float64

	// Enemy type for compatibility
	EnemyType EnemyType

	// Weapon cooldowns (tracked per weapon type)
	WeaponCooldowns map[WeaponType]float64
}

// NewConvexAIInput creates a new Convex-driven AI input provider
func NewConvexAIInput(scriptCode string, runner *WASMRunner) *ConvexAIInput {
	return &ConvexAIInput{
		scriptCode:      scriptCode,
		runner:          runner,
		scriptCooldown:  0.05, // Run script every 50ms (20 times per second)
		timeSinceExec:   0.0,
		EnemyType:       EnemyTypeRocket,
		WeaponCooldowns: make(map[WeaponType]float64),
	}
}

// SetEntity sets the entity reference for the AI
func (ai *ConvexAIInput) SetEntity(entity *Entity) {
	ai.entity = entity
}

// SetPlayer sets the player reference for the AI
func (ai *ConvexAIInput) SetPlayer(player *Entity) {
	ai.player = player
}

// SetWorld sets the world reference for the AI
func (ai *ConvexAIInput) SetWorld(world *World) {
	ai.world = world
}

// GetThrust returns forward thrust from the AI decision
func (ai *ConvexAIInput) GetThrust() float64 {
	return ai.lastDecision.Thrust
}

// GetRotation returns rotation from the AI decision
func (ai *ConvexAIInput) GetRotation() float64 {
	// If TargetAngle is set, calculate rotation to reach it
	if ai.lastDecision.TargetAngle != nil && ai.entity != nil {
		targetAngle := *ai.lastDecision.TargetAngle
		currentAngle := ai.entity.Rotation

		// Calculate angle difference
		diff := targetAngle - currentAngle

		// Normalize to [-π, π]
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}

		// Return rotation direction (-1 or 1 based on shortest path)
		if math.Abs(diff) < 0.05 {
			return 0 // Close enough
		}
		if diff > 0 {
			return 1.0 // Rotate clockwise
		}
		return -1.0 // Rotate counter-clockwise
	}

	return ai.lastDecision.RotationSpeed
}

// ShouldShoot returns whether the AI wants to shoot
func (ai *ConvexAIInput) ShouldShoot() bool {
	return ai.lastDecision.ShouldShoot
}

// HasTarget returns true if the AI has a valid target
func (ai *ConvexAIInput) HasTarget() bool {
	return ai.player != nil && ai.player.Active
}

// ResetWeaponCooldown resets cooldown for a specific weapon type
func (ai *ConvexAIInput) ResetWeaponCooldown(weaponType WeaponType) {
	if ai.WeaponCooldowns == nil {
		ai.WeaponCooldowns = make(map[WeaponType]float64)
	}
	ai.WeaponCooldowns[weaponType] = 0.0
}

// Update updates the AI state and executes the script if needed
func (ai *ConvexAIInput) Update(deltaTime float64) {
	ai.deltaTime = deltaTime
	ai.gameTime += deltaTime
	ai.timeSinceExec += deltaTime

	// Update weapon cooldowns
	for weaponType := range ai.WeaponCooldowns {
		ai.WeaponCooldowns[weaponType] += deltaTime
	}

	// Only run script at specified frequency
	if ai.timeSinceExec < ai.scriptCooldown {
		return
	}
	ai.timeSinceExec = 0

	// Build AI context
	if ai.entity == nil {
		return
	}

	ctx := BuildAIContext(ai.entity, ai.player, ai.world, deltaTime, ai.gameTime)

	// Execute script
	if ai.runner != nil && ai.scriptCode != "" {
		decision, err := ai.runner.ExecuteScript(ai.scriptCode, ctx)
		if err != nil {
			// Log error but don't crash
			fmt.Printf("Script execution error: %v\n", err)
			// Use default behavior on error
			ai.lastDecision = defaultAIDecision(ctx)
			return
		}
		ai.lastDecision = decision
	} else {
		// No script, use default behavior
		ai.lastDecision = defaultAIDecision(ctx)
	}
}

// defaultAIDecision provides a simple chase-player behavior as fallback
func defaultAIDecision(ctx AIContext) AIDecision {
	decision := AIDecision{
		Thrust:      1.0,
		ShouldShoot: false,
	}

	if ctx.PlayerActive {
		// Calculate angle to player
		angle := ctx.AngleToPlayer
		decision.TargetAngle = &angle

		// Shoot if close enough
		if ctx.DistanceToPlayer < 300 {
			decision.ShouldShoot = true
		}
	}

	return decision
}

// UpdateConvexAI updates a ConvexAIInput with current game state
// This should be called from the game loop before entity update
func UpdateConvexAI(ai *ConvexAIInput, entity *Entity, player *Entity, world *World, deltaTime float64) {
	ai.SetEntity(entity)
	ai.SetPlayer(player)
	ai.SetWorld(world)
	// Note: Update() is called via InputProvider interface, not here
}
