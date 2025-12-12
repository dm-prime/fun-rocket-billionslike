package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Game represents the main game state
type Game struct {
	world           *World
	collisionSystem *CollisionSystem
	renderer        *Renderer
	camera          *Camera
	config          Config

	// Player entity
	player *Entity

	// Projectile pool
	projectiles    []*Entity
	maxProjectiles int

	// Enemy spawn timer
	enemySpawnTimer float64
	enemySpawnRate  float64

	// Wave-based spawning
	waveNumber             int
	enemiesPerWave         int
	enemiesSpawnedThisWave int
	waveSpawnTimer         float64 // Time between enemy spawns within a wave
	waveCooldown           float64 // Time between waves

	// Player score
	score int

	// FPS tracking
	fps              float64
	fpsUpdateCounter int
	fpsUpdateTimer   float64

	// Performance profiling
	profiler *Profiler

	// FPS drop detection
	lastFPSDropTime time.Time
	fpsDropCooldown time.Duration

	// Last update time for delta time calculation
	lastUpdateTime time.Time
}

// NewGame creates a new game instance
func NewGame(config Config) *Game {
	world := NewWorld(config)
	collisionSystem := NewCollisionSystem(world)
	camera := NewCamera(float64(config.ScreenWidth), float64(config.ScreenHeight))
	renderer := NewRenderer(camera)

	game := &Game{
		world:                  world,
		collisionSystem:        collisionSystem,
		renderer:               renderer,
		camera:                 camera,
		config:                 config,
		maxProjectiles:         1000,
		projectiles:            make([]*Entity, 0, 1000),
		enemySpawnRate:         0.5, // Spawn enemy every 0.5 seconds (legacy, kept for compatibility)
		waveNumber:             1,
		enemiesPerWave:         10, // Start with 10 enemies per wave
		enemiesSpawnedThisWave: 0,
		waveSpawnTimer:         0.1, // Spawn enemies quickly within a wave (0.1 seconds apart)
		waveCooldown:           5.0, // 5 seconds between waves
		score:                  0,
		fps:                    60.0,
		fpsUpdateCounter:       0,
		fpsUpdateTimer:         0.0,
		profiler:               NewProfiler(),
		fpsDropCooldown:        10 * time.Second, // Don't trigger profiling more than once every 10 seconds
		lastUpdateTime:         time.Now(),
	}

	// Create player
	game.createPlayer()

	// Spawn initial wave of enemies
	game.enemiesPerWave = 10
	game.enemiesSpawnedThisWave = 0

	return game
}

// createPlayer creates the player entity
func (g *Game) createPlayer() {
	playerInput := NewPlayerInput()
	g.player = NewEntityWithShipType(
		g.config.WorldWidth/2,
		g.config.WorldHeight/2,
		EntityTypePlayer,
		ShipTypePlayer,
		playerInput,
	)
	g.player.Faction = FactionPlayer // Set player faction
	g.world.RegisterEntity(g.player)

	// Center camera on player
	g.camera.X = g.player.X
	g.camera.Y = g.player.Y
}

// respawnPlayer resets the entire game state
func (g *Game) respawnPlayer() {
	// Collect entities to remove (avoid modifying slice while iterating)
	enemiesToRemove := make([]*Entity, 0)
	for _, entity := range g.world.AllEntities {
		if entity.Type == EntityTypeEnemy {
			enemiesToRemove = append(enemiesToRemove, entity)
		}
	}

	// Remove enemies
	for _, entity := range enemiesToRemove {
		entity.Active = false
		g.world.UnregisterEntity(entity)
	}

	// Clear all projectiles
	for _, projectile := range g.projectiles {
		projectile.Active = false
		g.world.UnregisterEntity(projectile)
	}
	g.projectiles = g.projectiles[:0] // Clear slice but keep capacity

	// Reset player
	if g.player != nil {
		// Ensure player input is still set (reinitialize if needed)
		if g.player.Input == nil {
			g.player.Input = NewPlayerInput()
		}

		// Reset player position to center
		g.player.X = g.config.WorldWidth / 2
		g.player.Y = g.config.WorldHeight / 2

		// Reset velocity
		g.player.VX = 0
		g.player.VY = 0
		g.player.AngularVelocity = 0

		// Reset rotation
		g.player.Rotation = 0

		// Reset turret rotations
		if playerInput, ok := g.player.Input.(*PlayerInput); ok {
			playerInput.TurretRotations = make(map[int]float64)
			playerInput.TurretTargets = make(map[int]TurretTarget)
		}

		// Restore health
		g.player.Health = g.player.MaxHealth

		// Reset age
		g.player.Age = 0.0

		// Reset faction
		g.player.Faction = FactionPlayer

		// Ensure player is active
		g.player.Active = true

		// Ensure player is registered in world
		if !g.isPlayerRegistered() {
			g.world.RegisterEntity(g.player)
		} else {
			// Update cell membership
			g.world.UpdateEntityCell(g.player)
		}

		// Center camera on player
		g.camera.X = g.player.X
		g.camera.Y = g.player.Y
	}

	// Reset spawn timer and wave state
	g.enemySpawnTimer = 0
	g.waveNumber = 1
	g.enemiesPerWave = 10
	g.enemiesSpawnedThisWave = 0
	g.waveSpawnTimer = 0

	// Reset score
	g.score = 0
}

// isPlayerRegistered checks if the player is registered in the world
func (g *Game) isPlayerRegistered() bool {
	if g.player == nil {
		return false
	}
	for _, entity := range g.world.AllEntities {
		if entity == g.player {
			return true
		}
	}
	return false
}

// updatePlayerTargeting finds the nearest enemy for each turret and updates turret rotation to face it
// Each turret targets a different enemy to split fire
func (g *Game) updatePlayerTargeting(playerInput *PlayerInput, deltaTime float64) {
	if g.player == nil || !g.player.Active {
		// Clear all turret targets
		playerInput.TurretTargets = make(map[int]TurretTarget)
		return
	}

	shipConfig := GetShipTypeConfig(g.player.ShipType)
	playerFaction := GetEntityFaction(g.player)

	// Calculate ship rotation transforms once
	cosRot := math.Cos(g.player.Rotation)
	sinRot := math.Sin(g.player.Rotation)

	// Track which enemies are already targeted by other turrets
	targetedEnemies := make(map[*Entity]bool)

	// Use spatial partitioning to find nearby enemies instead of iterating all entities
	maxTargetRange := playerInput.MaxTargetRange
	candidates := g.world.GetEntitiesInRadius(g.player.X, g.player.Y, maxTargetRange*1.5) // Slightly larger radius to account for turret offsets

	// Process each turret separately
	for turretIndex, mount := range shipConfig.TurretMounts {
		if !mount.Active {
			continue
		}

		// Calculate turret position in world coordinates
		mountX := mount.OffsetX*cosRot - mount.OffsetY*sinRot
		mountY := mount.OffsetX*sinRot + mount.OffsetY*cosRot
		turretX := g.player.X + mountX
		turretY := g.player.Y + mountY

		// Find nearest enemy from this turret's position that isn't already targeted
		var nearestEnemy *Entity
		nearestDistanceSq := maxTargetRange * maxTargetRange // Use squared distance to avoid sqrt

		// Search through nearby entities instead of all entities
		for _, entity := range candidates {
			if !entity.Active || entity.Health <= 0 {
				continue
			}

			// Only target entities of opposite faction
			entityFaction := GetEntityFaction(entity)
			if entityFaction == playerFaction {
				continue // Skip friendly entities
			}

			// Skip enemies already targeted by other turrets
			if targetedEnemies[entity] {
				continue
			}

			// Calculate squared distance from turret position to enemy (avoid sqrt)
			dx := entity.X - turretX
			dy := entity.Y - turretY
			distanceSq := dx*dx + dy*dy

			if distanceSq < nearestDistanceSq {
				nearestDistanceSq = distanceSq
				nearestEnemy = entity
			}
		}

		// Update target and rotate turret
		if nearestEnemy != nil {
			// Mark this enemy as targeted
			targetedEnemies[nearestEnemy] = true

			// Calculate predictive aim target from this turret's position
			predictedX, predictedY := CalculatePredictiveAim(turretX, turretY, nearestEnemy)

			// Store predicted target position for this turret
			playerInput.TurretTargets[turretIndex] = TurretTarget{
				TargetX:   predictedX,
				TargetY:   predictedY,
				HasTarget: true,
			}

			// Calculate angle from turret to predicted target
			turretDx := predictedX - turretX
			turretDy := predictedY - turretY
			turretTargetRotation := math.Atan2(turretDy, turretDx)

			// Get current rotation for this turret (or initialize to ship rotation + mount angle)
			currentRotation := playerInput.GetTurretRotation(turretIndex)
			if currentRotation == 0.0 {
				currentRotation = g.player.Rotation + mount.Angle
			}

			// Smoothly rotate turret towards target
			maxTurretAngularVelocity := 8.0 // radians per second (faster than ship)
			newRotation := RotateTowardsTarget(
				currentRotation,
				turretTargetRotation,
				maxTurretAngularVelocity,
				deltaTime,
			)
			playerInput.TurretRotations[turretIndex] = newRotation
		} else {
			// No target for this turret
			playerInput.TurretTargets[turretIndex] = TurretTarget{HasTarget: false}
		}
	}
}

// spawnEnemy spawns a new enemy at a random position near the player
func (g *Game) spawnEnemy() {
	var x, y float64

	if g.player != nil && g.player.Active {
		// Spawn enemies around the player at a distance
		spawnDistance := 400.0 + rand.Float64()*200.0 // 400-600 pixels away
		angle := rand.Float64() * 2 * math.Pi
		x = g.player.X + math.Cos(angle)*spawnDistance
		y = g.player.Y + math.Sin(angle)*spawnDistance

		// Clamp to world bounds
		x = math.Max(0, math.Min(x, g.config.WorldWidth))
		y = math.Max(0, math.Min(y, g.config.WorldHeight))
	} else {
		// Fallback: spawn at edge of world
		side := rand.Intn(4)
		switch side {
		case 0: // Top
			x = rand.Float64() * g.config.WorldWidth
			y = 0
		case 1: // Right
			x = g.config.WorldWidth
			y = rand.Float64() * g.config.WorldHeight
		case 2: // Bottom
			x = rand.Float64() * g.config.WorldWidth
			y = g.config.WorldHeight
		case 3: // Left
			x = 0
			y = rand.Float64() * g.config.WorldHeight
		}
	}

	// Choose random enemy type
	enemyType := GetRandomEnemyType()

	aiInput := CreateEnemyAIWithType(enemyType)
	enemy := NewEntityWithShipType(x, y, EntityTypeEnemy, GetEnemyTypeConfig(enemyType).ShipType, aiInput)
	enemy.Faction = FactionEnemy // Explicitly set faction to enemy (regardless of ship type)
	g.world.RegisterEntity(enemy)
}

// spawnProjectile spawns a projectile from an entity using weapon types
// Fires from all active turrets
func (g *Game) spawnProjectile(entity *Entity) {
	shipConfig := GetShipTypeConfig(entity.ShipType)

	// Don't shoot if there are no turret mounts
	if len(shipConfig.TurretMounts) == 0 {
		return
	}

	// Calculate ship rotation transforms once
	cosRot := math.Cos(entity.Rotation)
	sinRot := math.Sin(entity.Rotation)

	// Fire from all active turrets (checking weapon cooldowns)
	for i := range shipConfig.TurretMounts {
		mount := &shipConfig.TurretMounts[i]
		if !mount.Active {
			continue
		}

		// Check weapon cooldown
		weaponConfig := GetWeaponConfig(mount.WeaponType)
		var timeSinceLastShot float64
		var hasBeenFired bool

		if playerInput, ok := entity.Input.(*PlayerInput); ok {
			if playerInput.WeaponCooldowns != nil {
				timeSinceLastShot, hasBeenFired = playerInput.WeaponCooldowns[mount.WeaponType]
			}
		} else if aiInput, ok := entity.Input.(*AIInput); ok {
			if aiInput.WeaponCooldowns != nil {
				timeSinceLastShot, hasBeenFired = aiInput.WeaponCooldowns[mount.WeaponType]
			}
		}

		if !weaponConfig.CanShoot(timeSinceLastShot, hasBeenFired) {
			continue // Skip this turret if weapon is on cooldown
		}

		// Transform mount offset from ship-local to world coordinates
		mountX := mount.OffsetX*cosRot - mount.OffsetY*sinRot
		mountY := mount.OffsetX*sinRot + mount.OffsetY*cosRot

		// Calculate turret mount position in world coordinates
		turretX := entity.X + mountX
		turretY := entity.Y + mountY

		// Use turret rotation for shooting direction (or ship rotation + mount angle for AI)
		var shootRotation float64
		if playerInput, ok := entity.Input.(*PlayerInput); ok {
			// Use per-turret rotation, fallback to ship rotation + mount angle if not set
			shootRotation = playerInput.GetTurretRotation(i)
			if shootRotation == 0.0 {
				shootRotation = entity.Rotation + mount.Angle
			}
			// Reset weapon cooldown after firing
			playerInput.ResetWeaponCooldown(mount.WeaponType)
		} else if aiInput, ok := entity.Input.(*AIInput); ok {
			shootRotation = entity.Rotation + mount.Angle
			// Reset weapon cooldown after firing
			aiInput.ResetWeaponCooldown(mount.WeaponType)
		} else {
			shootRotation = entity.Rotation + mount.Angle
		}

		// Spawn position is at the end of the barrel (turret position + barrel length in turret direction)
		spawnX := turretX + math.Cos(shootRotation)*mount.BarrelLength
		spawnY := turretY + math.Sin(shootRotation)*mount.BarrelLength

		// Spawn weapon projectile based on turret's weapon type
		g.spawnWeaponProjectile(mount.WeaponType, spawnX, spawnY, shootRotation, entity)
	}
}

// spawnWeaponProjectile spawns a projectile based on weapon type
func (g *Game) spawnWeaponProjectile(weaponType WeaponType, spawnX, spawnY, rotation float64, owner *Entity) {
	weaponConfig := GetWeaponConfig(weaponType)

	switch weaponType {
	case WeaponTypeBullet:
		g.spawnBullet(spawnX, spawnY, rotation, owner, weaponConfig)
	case WeaponTypeHomingMissile:
		g.spawnHomingMissile(spawnX, spawnY, rotation, owner, weaponConfig)
	default:
		// Fallback to bullet
		g.spawnBullet(spawnX, spawnY, rotation, owner, GetWeaponConfig(WeaponTypeBullet))
	}
}

// spawnBullet spawns a bullet projectile
func (g *Game) spawnBullet(spawnX, spawnY, rotation float64, owner *Entity, weaponConfig WeaponConfig) {
	if len(g.projectiles) >= g.maxProjectiles {
		// Reuse oldest projectile
		projectile := g.projectiles[0]
		g.projectiles = g.projectiles[1:]
		g.world.UnregisterEntity(projectile)
		projectile.Reset()

		projectile.X = spawnX
		projectile.Y = spawnY
		projectile.Active = true
		projectile.Health = weaponConfig.Damage
		projectile.Type = EntityTypeProjectile
		projectile.Radius = weaponConfig.Radius
		projectile.Input = nil                       // Projectiles don't need input
		projectile.Age = 0.0                         // Reset age
		projectile.Owner = owner                     // Track who fired this projectile
		projectile.Faction = GetEntityFaction(owner) // Inherit faction from owner

		// Set velocity based on shoot rotation
		projectile.VX = math.Cos(rotation) * weaponConfig.ProjectileSpeed
		projectile.VY = math.Sin(rotation) * weaponConfig.ProjectileSpeed
		projectile.Rotation = rotation // Set projectile rotation to match direction

		g.world.RegisterEntity(projectile)
		g.projectiles = append(g.projectiles, projectile)
	} else {
		// Create new projectile
		projectile := NewEntity(spawnX, spawnY, weaponConfig.Radius, EntityTypeProjectile, nil)
		projectile.Health = weaponConfig.Damage
		projectile.MaxHealth = weaponConfig.Damage
		projectile.Age = 0.0                         // Initialize age
		projectile.Owner = owner                     // Track who fired this projectile
		projectile.Faction = GetEntityFaction(owner) // Inherit faction from owner

		// Set velocity based on shoot rotation
		projectile.VX = math.Cos(rotation) * weaponConfig.ProjectileSpeed
		projectile.VY = math.Sin(rotation) * weaponConfig.ProjectileSpeed
		projectile.Rotation = rotation // Set projectile rotation to match direction

		g.world.RegisterEntity(projectile)
		g.projectiles = append(g.projectiles, projectile)
	}
}

// spawnHomingMissile spawns a homing enemy (missile) that targets the opposite faction
func (g *Game) spawnHomingMissile(spawnX, spawnY, rotation float64, owner *Entity, weaponConfig WeaponConfig) {
	if owner == nil {
		return
	}

	// Get faction directly from owner entity (faction is set at spawn time)
	ownerFaction := owner.Faction

	// Spawn homing entity with same faction as owner
	homingEnemyType := EnemyTypeRocket
	homingShipType := GetEnemyTypeConfig(homingEnemyType).ShipType

	homingAI := CreateEnemyAIWithType(homingEnemyType)
	homingEnemy := NewEntityWithShipType(spawnX, spawnY, EntityTypeEnemy, homingShipType, homingAI)
	homingEnemy.Faction = ownerFaction // Inherit faction from owner
	homingEnemy.NoCollision = true     // Homing rockets don't collide with other entities (except targets)
	homingEnemy.Lifetime = weaponConfig.Lifetime // Set lifetime for auto-detonation

	// Give the homing enemy initial velocity in the shooting direction
	homingEnemy.VX = math.Cos(rotation) * weaponConfig.InitialVelocity
	homingEnemy.VY = math.Sin(rotation) * weaponConfig.InitialVelocity
	homingEnemy.Rotation = rotation

	g.world.RegisterEntity(homingEnemy)
}

// createDestroyedIndicator creates a visual indicator at the specified position
// that shows a missile was destroyed, colored by the faction
func (g *Game) createDestroyedIndicator(x, y float64, faction Faction) {
	indicator := NewEntity(x, y, 8.0, EntityTypeDestroyedIndicator, nil)
	indicator.Faction = faction
	indicator.Active = true
	indicator.Health = 1.0 // Small health value so it renders
	indicator.MaxHealth = 1.0
	indicator.Lifetime = 1.0 // Show for 1 second
	indicator.Age = 0.0
	indicator.NoCollision = true // Don't collide with anything
	g.world.RegisterEntity(indicator)
}

// Update updates the game state
func (g *Game) Update() error {
	// Calculate delta time
	now := time.Now()
	deltaTime := now.Sub(g.lastUpdateTime).Seconds()
	g.lastUpdateTime = now

	// Clamp delta time to prevent large jumps
	if deltaTime > 0.1 {
		deltaTime = 0.1
	}

	// Update FPS calculation (update every 0.5 seconds)
	g.fpsUpdateTimer += deltaTime
	g.fpsUpdateCounter++
	if g.fpsUpdateTimer >= 0.5 {
		if g.fpsUpdateCounter > 0 {
			g.fps = float64(g.fpsUpdateCounter) / g.fpsUpdateTimer
		}
		
		// Detect FPS drops below 60 FPS
		if g.fps < 60.0 && time.Since(g.lastFPSDropTime) >= g.fpsDropCooldown {
			g.lastFPSDropTime = time.Now()
			
			// Generate reason string with context
			entityCount := len(g.world.AllEntities)
			projectileCount := len(g.projectiles)
			reason := fmt.Sprintf("fps%.0f-entities%d-projectiles%d", g.fps, entityCount, projectileCount)
			
			// Trigger performance capture
			err := g.profiler.CaptureProfile(reason)
			if err != nil {
				// Silently ignore cooldown errors, but log other errors
				if !strings.Contains(err.Error(), "cooldown") {
					fmt.Printf("Failed to capture profile: %v\n", err)
				}
			} else {
				fmt.Printf("FPS drop detected (%.0f FPS). Capturing performance profile...\n", g.fps)
			}
		}
		
		g.fpsUpdateCounter = 0
		g.fpsUpdateTimer = 0.0
	}

	// Update player input
	if g.player != nil && g.player.Input != nil {
		g.player.Input.Update(deltaTime)

		// Check for respawn
		if playerInput, ok := g.player.Input.(*PlayerInput); ok {
			if playerInput.ShouldRespawn() {
				g.respawnPlayer()
			}

			// Update player target acquisition AI
			g.updatePlayerTargeting(playerInput, deltaTime)
		}
	}

	// Update all entities
	for _, entity := range g.world.AllEntities {
		if !entity.Active {
			continue
		}

		// Update input/AI
		if entity.Input != nil {
			entity.Input.Update(deltaTime)

			// Update AI if it's an enemy
			if entity.Type == EntityTypeEnemy {
				if aiInput, ok := entity.Input.(*AIInput); ok {
					UpdateAI(aiInput, entity, g.player, g.world, deltaTime)
				}
			}
		}

		// Update entity
		entity.Update(deltaTime)

		// Check lifetime for homing missiles (auto-detonate after lifetime expires)
		if entity.Lifetime > 0 && entity.Age >= entity.Lifetime {
			// Lifetime expired - detonate the missile
			if entity.ShipType == ShipTypeHomingSuicide {
				// Create destroyed indicator at missile position
				g.createDestroyedIndicator(entity.X, entity.Y, entity.Faction)
				entity.Active = false
				entity.Health = 0
			}
		}

		// Handle shooting
		if entity.Input != nil && entity.Input.ShouldShoot() {
			if entity.Type == EntityTypePlayer || entity.Type == EntityTypeEnemy {
				g.spawnProjectile(entity)
				// Reset shoot cooldown for AI
				if aiInput, ok := entity.Input.(*AIInput); ok {
					aiInput.TimeSinceLastShot = 0
				}
			}
		}

		// Update entity cell membership
		g.collisionSystem.MoveEntity(entity)

		// Remove dead entities and expired destroyed indicators
		if entity.Health <= 0 || (entity.Type == EntityTypeDestroyedIndicator && entity.Lifetime > 0 && entity.Age >= entity.Lifetime) {
			// Award score if enemy was destroyed by player
			if entity.Type == EntityTypeEnemy {
				// Check if this enemy was destroyed by player (via projectile or collision)
				// We'll check this by looking at recent damage sources
				// For now, award score for any enemy death (can be refined later)
				shipConfig := GetShipTypeConfig(entity.ShipType)
				g.score += shipConfig.Score
			}
			entity.Active = false
			if entity.Type == EntityTypeProjectile {
				// Remove projectile from list
				for i, p := range g.projectiles {
					if p == entity {
						g.projectiles = append(g.projectiles[:i], g.projectiles[i+1:]...)
						break
					}
				}
			}
			g.world.UnregisterEntity(entity)
		}

		// Remove projectiles that are out of bounds
		if entity.Type == EntityTypeProjectile {
			if entity.X < 0 || entity.X > g.config.WorldWidth ||
				entity.Y < 0 || entity.Y > g.config.WorldHeight {
				entity.Active = false
				g.world.UnregisterEntity(entity)
			}
		}
	}

	// Check collisions
	g.collisionSystem.CheckCollisions()

	// Update camera to follow player
	if g.player != nil && g.player.Active {
		// Smooth camera follow
		dx := g.player.X - g.camera.X
		dy := g.player.Y - g.camera.Y
		g.camera.X += dx * 0.1
		g.camera.Y += dy * 0.1
	}

	// Wave-based enemy spawning
	if g.enemiesSpawnedThisWave < g.enemiesPerWave {
		// Still spawning enemies for current wave
		g.waveSpawnTimer += deltaTime
		if g.waveSpawnTimer >= 0.1 { // Spawn every 0.1 seconds within wave
			g.waveSpawnTimer = 0
			g.spawnEnemy()
			g.enemiesSpawnedThisWave++
		}
	} else {
		// Wave complete, wait for cooldown before next wave
		g.enemySpawnTimer += deltaTime
		if g.enemySpawnTimer >= g.waveCooldown {
			g.enemySpawnTimer = 0
			// Start next wave with +1 enemy
			g.waveNumber++
			g.enemiesPerWave++
			g.enemiesSpawnedThisWave = 0
			g.waveSpawnTimer = 0
		}
	}

	return nil
}

// Draw renders the game
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{20, 20, 40, 255}) // Dark blue background
	g.renderer.Render(screen, g.world, g.player, g.score, g.fps)
}

// Layout returns the game's screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.config.ScreenWidth, g.config.ScreenHeight
}
