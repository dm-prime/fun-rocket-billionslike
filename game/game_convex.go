package game

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// ConvexGame extends the base Game with Convex-driven AI scripts
type ConvexGame struct {
	*Game // Embed the base game

	// Convex client for fetching AI scripts
	convexClient *ConvexClient

	// WASM runner for executing scripts
	wasmRunner *WASMRunner

	// Cached AI scripts (name -> code)
	aiScripts map[string]string

	// Script-driven enemies
	scriptedEnemies []*Entity

	// Script refresh timer
	scriptRefreshTimer  float64
	scriptRefreshPeriod float64

	// Connection status
	convexConnected bool
	lastError       string
}

// NewConvexGame creates a new Convex-enabled game instance
func NewConvexGame(config Config, convexURL string) (*ConvexGame, error) {
	// Create base game
	baseGame := NewGame(config)

	// Create WASM runner
	runner, err := NewWASMRunner()
	if err != nil {
		return nil, fmt.Errorf("failed to create WASM runner: %w", err)
	}

	// Create Convex client
	var client *ConvexClient
	var connected bool
	if convexURL != "" {
		client = NewConvexClient(convexURL)
		connected = true
	}

	return &ConvexGame{
		Game:                baseGame,
		convexClient:        client,
		wasmRunner:          runner,
		aiScripts:           make(map[string]string),
		scriptedEnemies:     make([]*Entity, 0),
		scriptRefreshPeriod: 30.0, // Refresh scripts every 30 seconds
		convexConnected:     connected,
	}, nil
}

// Close releases resources
func (g *ConvexGame) Close() error {
	if g.wasmRunner != nil {
		return g.wasmRunner.Close()
	}
	return nil
}

// RefreshScripts fetches the latest AI scripts from Convex
func (g *ConvexGame) RefreshScripts() error {
	if g.convexClient == nil {
		return fmt.Errorf("convex client not initialized")
	}

	scripts, err := g.convexClient.ListAIScripts()
	if err != nil {
		g.lastError = err.Error()
		return err
	}

	// Update cached scripts
	for _, script := range scripts {
		g.aiScripts[script.Name] = script.Code
	}

	g.lastError = ""
	return nil
}

// GetScript returns a cached script by name
func (g *ConvexGame) GetScript(name string) (string, bool) {
	code, ok := g.aiScripts[name]
	return code, ok
}

// SpawnScriptedEnemy spawns an enemy controlled by a script
func (g *ConvexGame) SpawnScriptedEnemy(scriptName string) *Entity {
	code, ok := g.GetScript(scriptName)
	if !ok {
		fmt.Printf("Script '%s' not found\n", scriptName)
		return nil
	}

	// Spawn position near player
	var x, y float64
	if g.player != nil && g.player.Active {
		spawnDistance := 400.0 + rand.Float64()*200.0
		angle := rand.Float64() * 2 * math.Pi
		x = g.player.X + math.Cos(angle)*spawnDistance
		y = g.player.Y + math.Sin(angle)*spawnDistance

		// Clamp to world bounds
		x = math.Max(0, math.Min(x, g.config.WorldWidth))
		y = math.Max(0, math.Min(y, g.config.WorldHeight))
	} else {
		x = rand.Float64() * g.config.WorldWidth
		y = rand.Float64() * g.config.WorldHeight
	}

	// Create Convex AI input
	aiInput := NewConvexAIInput(code, g.wasmRunner)

	// Create enemy entity with shooter ship type (has more health, can shoot)
	enemy := NewEntityWithShipType(x, y, EntityTypeEnemy, ShipTypeShooter, aiInput)
	enemy.Faction = FactionEnemy

	// Register with world
	g.world.RegisterEntity(enemy)
	g.scriptedEnemies = append(g.scriptedEnemies, enemy)

	return enemy
}

// SpawnScriptedEnemyWithType spawns an enemy with a specific ship type
func (g *ConvexGame) SpawnScriptedEnemyWithType(scriptName string, shipType ShipType) *Entity {
	code, ok := g.GetScript(scriptName)
	if !ok {
		fmt.Printf("Script '%s' not found\n", scriptName)
		return nil
	}

	// Spawn position near player
	var x, y float64
	if g.player != nil && g.player.Active {
		spawnDistance := 400.0 + rand.Float64()*200.0
		angle := rand.Float64() * 2 * math.Pi
		x = g.player.X + math.Cos(angle)*spawnDistance
		y = g.player.Y + math.Sin(angle)*spawnDistance

		x = math.Max(0, math.Min(x, g.config.WorldWidth))
		y = math.Max(0, math.Min(y, g.config.WorldHeight))
	} else {
		x = rand.Float64() * g.config.WorldWidth
		y = rand.Float64() * g.config.WorldHeight
	}

	aiInput := NewConvexAIInput(code, g.wasmRunner)
	enemy := NewEntityWithShipType(x, y, EntityTypeEnemy, shipType, aiInput)
	enemy.Faction = FactionEnemy

	g.world.RegisterEntity(enemy)
	g.scriptedEnemies = append(g.scriptedEnemies, enemy)

	return enemy
}

// Update updates the game state (overrides base Game.Update)
func (g *ConvexGame) Update() error {
	// Calculate delta time
	now := time.Now()
	deltaTime := now.Sub(g.lastUpdateTime).Seconds()
	g.lastUpdateTime = now

	if deltaTime > 0.1 {
		deltaTime = 0.1
	}

	// Refresh scripts periodically
	g.scriptRefreshTimer += deltaTime
	if g.scriptRefreshTimer >= g.scriptRefreshPeriod {
		g.scriptRefreshTimer = 0
		go func() {
			if err := g.RefreshScripts(); err != nil {
				fmt.Printf("Failed to refresh scripts: %v\n", err)
			}
		}()
	}

	// Update Convex AI inputs for scripted enemies
	for _, enemy := range g.scriptedEnemies {
		if !enemy.Active {
			continue
		}
		if convexAI, ok := enemy.Input.(*ConvexAIInput); ok {
			UpdateConvexAI(convexAI, enemy, g.player, g.world, deltaTime)
		}
	}

	// Clean up inactive scripted enemies
	activeEnemies := make([]*Entity, 0, len(g.scriptedEnemies))
	for _, enemy := range g.scriptedEnemies {
		if enemy.Active {
			activeEnemies = append(activeEnemies, enemy)
		}
	}
	g.scriptedEnemies = activeEnemies

	// Call base game update
	return g.Game.Update()
}

// Draw renders the game (overrides base Game.Draw)
func (g *ConvexGame) Draw(screen *ebiten.Image) {
	// Call base draw
	g.Game.Draw(screen)

	// Draw Convex mode indicator
	statusColor := color.RGBA{0, 255, 0, 255} // Green = connected
	statusText := "CONVEX MODE"
	if !g.convexConnected {
		statusColor = color.RGBA{255, 165, 0, 255} // Orange = no URL
		statusText = "CONVEX MODE (offline)"
	}
	if g.lastError != "" {
		statusColor = color.RGBA{255, 0, 0, 255} // Red = error
		statusText = fmt.Sprintf("CONVEX ERROR: %s", g.lastError)
	}

	// Draw status in top-right corner
	ebitenutil.DebugPrintAt(screen, statusText, g.config.ScreenWidth-200, 10)
	_ = statusColor // Would use for colored text if available

	// Draw script count
	scriptInfo := fmt.Sprintf("Scripts: %d | Scripted enemies: %d", len(g.aiScripts), len(g.scriptedEnemies))
	ebitenutil.DebugPrintAt(screen, scriptInfo, g.config.ScreenWidth-250, 30)
}

// LoadAndSpawnScriptedEnemy fetches a script by name and spawns an enemy
func (g *ConvexGame) LoadAndSpawnScriptedEnemy(scriptName string) (*Entity, error) {
	// Try to fetch script if not cached
	if _, ok := g.aiScripts[scriptName]; !ok {
		if g.convexClient == nil {
			return nil, fmt.Errorf("convex client not initialized")
		}
		code, err := g.convexClient.FetchAIScript(scriptName)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch script '%s': %w", scriptName, err)
		}
		g.aiScripts[scriptName] = code
	}

	enemy := g.SpawnScriptedEnemy(scriptName)
	if enemy == nil {
		return nil, fmt.Errorf("failed to spawn scripted enemy")
	}

	return enemy, nil
}

// AddScript adds a script directly (useful for testing)
func (g *ConvexGame) AddScript(name, code string) error {
	// Validate script first
	if err := g.wasmRunner.ValidateScript(code); err != nil {
		return fmt.Errorf("invalid script: %w", err)
	}
	g.aiScripts[name] = code
	return nil
}

// GetExampleScript returns an example AI script for testing
func GetExampleScript() string {
	return `
function decide(ctx) {
    // Simple chase-player AI
    var dx = ctx.playerX - ctx.entityX;
    var dy = ctx.playerY - ctx.entityY;
    var dist = Math.sqrt(dx*dx + dy*dy);
    
    // Calculate angle to player
    var targetAngle = Math.atan2(dy, dx);
    
    return {
        moveX: dx / dist,
        moveY: dy / dist,
        thrust: 1.0,
        targetAngle: targetAngle,
        shouldShoot: dist < 300,
        rotationSpeed: 0
    };
}
`
}

// GetCircleScript returns an AI script that circles around the player
func GetCircleScript() string {
	return `
function decide(ctx) {
    var dx = ctx.playerX - ctx.entityX;
    var dy = ctx.playerY - ctx.entityY;
    var dist = Math.sqrt(dx*dx + dy*dy);
    
    // Desired orbit distance
    var orbitDist = 200;
    
    // Angle to player
    var angleToPlayer = Math.atan2(dy, dx);
    
    // If too far, move toward player; if too close, move away
    var thrust = 0;
    if (dist > orbitDist + 50) {
        thrust = 1.0;
    } else if (dist < orbitDist - 50) {
        thrust = -0.5;
    } else {
        thrust = 0.3; // Maintain orbit
    }
    
    // Orbit perpendicular to player direction
    var orbitAngle = angleToPlayer + Math.PI / 2;
    
    return {
        thrust: thrust,
        targetAngle: orbitAngle,
        shouldShoot: dist < 300 && ctx.playerActive,
        rotationSpeed: 0
    };
}
`
}
