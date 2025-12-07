package main

import (
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func newGame() *Game {
	rand.Seed(time.Now().UnixNano())

	g := &Game{
		ships:            make(map[EntityID]*Ship),
		rocks:            make(map[EntityID]*Rock),
		bullets:          make(map[EntityID]*Bullet),
		dust:             make([]dust, dustCount),
		radarTrails:      make(map[EntityID][]RadarTrailPoint),
		radarTrailTimers: make(map[EntityID]float64),
		npcStates:        make(map[EntityID]NPCState),
		npcInputs:        make(map[EntityID]ShipInput),
		gameTime:         0,
		gameOver:         false,
		prevRestartKey:   false,
		prevSpaceKey:     false,
		waveSpawnTimer:   0,
		waveNumber:       0,
	}
	g.initFactions()

	// Create player ship
	playerShip := NewShip(
		vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5},
		vec2{0, 0},
		0,
		"Union",
		true,
	)
	g.ships[playerShip.ID()] = playerShip
	g.playerID = playerShip.ID()
	g.initTurretPoints(playerShip)

	// Create a few demo NPC ships
	demoShips := []*Ship{
		NewShip(
			vec2{float64(screenWidth)*0.5 + 120, float64(screenHeight)*0.5 - 60},
			vec2{30, -10},
			math.Pi*0.25,
			"Raiders",
			false,
		),
		NewShip(
			vec2{float64(screenWidth)*0.5 - 160, float64(screenHeight)*0.5 + 90},
			vec2{-20, 25},
			-math.Pi*0.5,
			"Raiders",
			false,
		),
		NewShip(
			vec2{float64(screenWidth)*0.5 + 220, float64(screenHeight)*0.5 + 40},
			vec2{15, 5},
			math.Pi*0.15,
			"Traders",
			false,
		),
	}

	// Add demo ships and initialize turrets
	for _, ship := range demoShips {
		g.ships[ship.ID()] = ship
		g.initTurretPoints(ship)
	}

	// Seed dust in a square around the player so rotated views stay filled.
	initialSpan := math.Hypot(float64(screenWidth), float64(screenHeight)) * dustSpanMultiplier
	halfSpan := initialSpan * 0.5
	for i := range g.dust {
		g.dust[i] = dust{
			pos: vec2{
				x: playerShip.pos.x + rand.Float64()*initialSpan - halfSpan,
				y: playerShip.pos.y + rand.Float64()*initialSpan - halfSpan,
			},
			speed:  0.5 + rand.Float64()*1.0, // Speed multiplier from 0.5x to 1.5x
			radius: 1,
		}
	}

	// Initialize rock spawn timer
	g.rockSpawnTimer = 0

	// Initialize wave spawn timer (start first wave after a short delay)
	g.waveSpawnTimer = waveSpawnInterval * 0.5

	return g
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0
	g.gameTime += dt

	g.handleInput()

	// Check if player is dead
	player := g.PlayerShip()
	if player != nil && player.health <= 0 && !g.gameOver {
		g.gameOver = true
	}

	// Don't update game state when game over (except input handling)
	if g.gameOver {
		return nil
	}

	if player == nil {
		// Player ship doesn't exist - game over
		g.gameOver = true
		return nil
	}

	// Update all ships using unified physics system
	for id, ship := range g.ships {
		// Skip dead ships (they'll be removed later)
		if ship.health <= 0 {
			continue
		}

		var input ShipInput

		if ship.isPlayer {
			// Player: read keyboard input
			input = getPlayerInput()
			// Handle player shooting
			g.handlePlayerShooting(ship)
		} else {
			// NPC: generate input from AI state machine
			input = g.updateNPCStateMachine(ship, player, dt)
			// Store NPC input for predictive trail rendering
			g.npcInputs[id] = input
		}

		// Apply physics using unified system
		g.updatePhysics(ship, input, dt)

		// Update turret firing for NPCs
		g.updateTurretFiring(ship, dt)
	}

	// Update all rocks (just position updates)
	for _, rock := range g.rocks {
		if rock.health <= 0 {
			continue
		}
		rock.pos.x += rock.vel.x * dt
		rock.pos.y += rock.vel.y * dt
	}

	// Update bullets
	g.updateBullets(dt)

	// Check and handle all collisions using unified collision system
	collisionSys := NewCollisionSystem(g)
	collisionSys.CheckAndHandleCollisions(dt)

	// Remove dead entities
	g.removeDeadEntities()

	// Manage rocks: despawn far ones, spawn new ones near path
	g.manageRocks(player, dt)

	// Spawn enemy waves
	g.spawnEnemyWaves(player, dt)

	g.updateDust(dt, player)

	// Update radar trails
	g.updateRadarTrails(dt, player)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colorBackground)

	// Draw game over screen if game is over
	if g.gameOver {
		g.drawGameOver(screen)
		return
	}

	player := g.PlayerShip()
	if player == nil {
		return // No player, nothing to draw
	}

	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}

	// Draw dust (already positioned relative to ship movement)
	for _, d := range g.dust {
		offset := vec2{d.pos.x - player.pos.x, d.pos.y - player.pos.y}
		rot := rotatePoint(offset, -player.angle)
		drawCircle(screen, screenCenter.x+rot.x, screenCenter.y+rot.y, d.radius, colorDust)
	}

	// Draw all ships
	for _, ship := range g.ships {
		// Position relative to player so camera is centered on player ship.
		offsetX := ship.pos.x - player.pos.x
		offsetY := ship.pos.y - player.pos.y
		// Rotate the world around the player so the player stays "upright".
		rotated := rotatePoint(vec2{offsetX, offsetY}, -player.angle)
		shipScreenX := screenCenter.x + rotated.x
		shipScreenY := screenCenter.y + rotated.y
		renderAngle := ship.angle - player.angle
		if ship.isPlayer {
			renderAngle = 0
		}
		velRender := rotatePoint(vec2{ship.vel.x, ship.vel.y}, -player.angle)
		g.drawShip(screen, ship, shipScreenX, shipScreenY, renderAngle, velRender, player)
	}

	// Draw all rocks
	for _, rock := range g.rocks {
		offsetX := rock.pos.x - player.pos.x
		offsetY := rock.pos.y - player.pos.y
		rotated := rotatePoint(vec2{offsetX, offsetY}, -player.angle)
		rockScreenX := screenCenter.x + rotated.x
		rockScreenY := screenCenter.y + rotated.y
		g.drawRock(screen, rock, rockScreenX, rockScreenY, player)
	}

	g.drawBullets(screen, player)
	g.drawOffscreenIndicators(screen, player)
	g.drawHUD(screen, player)
	g.drawRadar(screen, player)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Capture screen size on first layout call
	if !g.initialized && outsideWidth > 0 && outsideHeight > 0 {
		screenWidth = outsideWidth
		screenHeight = outsideHeight
		g.initialized = true
	}
	return screenWidth, screenHeight
}

// spawnEnemyWaves handles periodic spawning of enemy waves
func (g *Game) spawnEnemyWaves(player *Ship, dt float64) {
	g.waveSpawnTimer += dt

	if g.waveSpawnTimer >= waveSpawnInterval {
		g.waveSpawnTimer = 0
		g.waveNumber++

		// Calculate number of enemies for this wave (increases over time)
		numEnemies := enemiesPerWave + int(float64(g.waveNumber)*waveSizeIncrease)
		if numEnemies > 10 {
			numEnemies = 10 // Cap at 10 enemies per wave
		}

		// Spawn enemies around the player
		for i := 0; i < numEnemies; i++ {
			g.spawnEnemy(player)
		}
	}
}

// spawnEnemy creates a single enemy ship at a distance from the player
func (g *Game) spawnEnemy(player *Ship) {
	// Choose a random angle around the player
	angle := rand.Float64() * 2 * math.Pi

	// Spawn at a distance from player
	spawnX := player.pos.x + math.Cos(angle)*waveSpawnDistance
	spawnY := player.pos.y + math.Sin(angle)*waveSpawnDistance

	// Calculate angle toward player
	dx := player.pos.x - spawnX
	dy := player.pos.y - spawnY
	targetAngle := math.Atan2(dx, -dy)

	// Add some random variation to the angle
	angleVariation := (rand.Float64() - 0.5) * math.Pi * 0.3 // ±27 degrees
	targetAngle += angleVariation

	// Initial velocity toward player (with some randomness)
	speed := 50.0 + rand.Float64()*50.0 // 50-100 px/s
	velX := math.Sin(targetAngle) * speed
	velY := -math.Cos(targetAngle) * speed

	// Create enemy ship using factory
	enemy := NewShip(
		vec2{x: spawnX, y: spawnY},
		vec2{x: velX, y: velY},
		targetAngle,
		"Raiders", // Enemy faction
		false,
	)

	// Initialize turret points
	g.initTurretPoints(enemy)

	// Add to ships
	g.ships[enemy.ID()] = enemy

	// Initialize NPC state for this ship (start in Pursue state)
	g.setNPCState(enemy.ID(), NPCStatePursue)
}

// restart resets the game to initial state
func (g *Game) restart() {
	// Reset game state by creating a new game
	newG := newGame()

	// Copy over the new state
	g.ships = newG.ships
	g.rocks = newG.rocks
	g.bullets = newG.bullets
	g.playerID = newG.playerID
	g.dust = newG.dust
	g.factionColors = newG.factionColors
	g.alliances = newG.alliances
	g.radarTrails = newG.radarTrails
	g.radarTrailTimers = newG.radarTrailTimers
	g.npcStates = newG.npcStates
	g.npcInputs = newG.npcInputs
	g.rockSpawnTimer = newG.rockSpawnTimer
	g.gameTime = 0
	g.gameOver = false
	g.prevRestartKey = false
	g.prevSpaceKey = false
	g.waveSpawnTimer = waveSpawnInterval * 0.5
	g.waveNumber = 0
}

func main() {
	// Get current monitor size
	ebiten.SetFullscreen(true)
	monitorWidth, monitorHeight := ebiten.ScreenSizeInFullscreen()

	screenWidth = monitorWidth
	screenHeight = monitorHeight

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Pocket Rocket - Ebiten Demo")
	ebiten.SetTPS(60)

	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
