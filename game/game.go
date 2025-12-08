package game

import (
	"image/color"
	"math"
	"math/rand"
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
	projectiles []*Entity
	maxProjectiles int

	// Enemy spawn timer
	enemySpawnTimer float64
	enemySpawnRate  float64

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
		world:           world,
		collisionSystem: collisionSystem,
		renderer:        renderer,
		camera:          camera,
		config:          config,
		maxProjectiles:  1000,
		projectiles:     make([]*Entity, 0, 1000),
		enemySpawnRate:  0.5, // Spawn enemy every 0.5 seconds
		lastUpdateTime:  time.Now(),
	}

	// Create player
	game.createPlayer()

	// Spawn initial enemies
	for i := 0; i < 10; i++ {
		game.spawnEnemy()
	}

	return game
}

// createPlayer creates the player entity
func (g *Game) createPlayer() {
	playerInput := NewPlayerInput()
	g.player = NewEntity(
		g.config.WorldWidth/2,
		g.config.WorldHeight/2,
		15.0, // radius
		EntityTypePlayer,
		playerInput,
	)
	g.player.MaxHealth = 100.0
	g.player.Health = 100.0
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

		// Restore health
		g.player.Health = g.player.MaxHealth

		// Reset age
		g.player.Age = 0.0

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

	// Reset spawn timer
	g.enemySpawnTimer = 0

	// Spawn initial enemies
	for i := 0; i < 10; i++ {
		g.spawnEnemy()
	}
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
	config := GetEnemyTypeConfig(enemyType)
	
	aiInput := CreateEnemyAIWithType(enemyType)
	enemy := NewEntity(x, y, config.Radius, EntityTypeEnemy, aiInput)
	enemy.MaxHealth = config.Health
	enemy.Health = config.Health
	g.world.RegisterEntity(enemy)
}

// spawnProjectile spawns a projectile from an entity
func (g *Game) spawnProjectile(entity *Entity) {
	if len(g.projectiles) >= g.maxProjectiles {
		// Reuse oldest projectile
		projectile := g.projectiles[0]
		g.projectiles = g.projectiles[1:]
		g.world.UnregisterEntity(projectile)
		projectile.Reset()
		// Spawn projectile slightly in front of entity to avoid immediate collision
		spawnOffset := entity.Radius + 8.0
		projectile.X = entity.X + math.Cos(entity.Rotation)*spawnOffset
		projectile.Y = entity.Y + math.Sin(entity.Rotation)*spawnOffset
		projectile.Active = true
		projectile.Health = 1.0
		projectile.Type = EntityTypeProjectile
		projectile.Radius = 5.0 // Make projectiles more visible
		projectile.Input = nil // Projectiles don't need input
		projectile.Age = 0.0 // Reset age

		// Set velocity based on entity rotation
		speed := 500.0
		projectile.VX = math.Cos(entity.Rotation) * speed
		projectile.VY = math.Sin(entity.Rotation) * speed
		projectile.Rotation = entity.Rotation // Set projectile rotation to match direction

		g.world.RegisterEntity(projectile)
		g.projectiles = append(g.projectiles, projectile)
	} else {
		// Create new projectile
		// Spawn projectile slightly in front of entity to avoid immediate collision
		spawnOffset := entity.Radius + 8.0
		spawnX := entity.X + math.Cos(entity.Rotation)*spawnOffset
		spawnY := entity.Y + math.Sin(entity.Rotation)*spawnOffset
		
		projectile := NewEntity(spawnX, spawnY, 5.0, EntityTypeProjectile, nil) // Make projectiles more visible
		projectile.Health = 1.0
		projectile.MaxHealth = 1.0
		projectile.Age = 0.0 // Initialize age

		// Set velocity based on entity rotation
		speed := 500.0
		projectile.VX = math.Cos(entity.Rotation) * speed
		projectile.VY = math.Sin(entity.Rotation) * speed
		projectile.Rotation = entity.Rotation // Set projectile rotation to match direction

		g.world.RegisterEntity(projectile)
		g.projectiles = append(g.projectiles, projectile)
	}
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

	// Update player input
	if g.player != nil && g.player.Input != nil {
		g.player.Input.Update(deltaTime)

		// Check for respawn
		if playerInput, ok := g.player.Input.(*PlayerInput); ok {
			if playerInput.ShouldRespawn() {
				g.respawnPlayer()
			}
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
					UpdateAI(aiInput, entity, g.player, deltaTime)
				}
			}
		}

		// Update entity
		entity.Update(deltaTime)

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

		// Remove dead entities
		if entity.Health <= 0 {
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

	// Spawn enemies
	g.enemySpawnTimer += deltaTime
	if g.enemySpawnTimer >= g.enemySpawnRate {
		g.enemySpawnTimer = 0
		g.spawnEnemy()
	}

	return nil
}

// Draw renders the game
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{20, 20, 40, 255}) // Dark blue background
	g.renderer.Render(screen, g.world)
}

// Layout returns the game's screen size
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.config.ScreenWidth, g.config.ScreenHeight
}

