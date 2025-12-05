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
		dust:             make([]dust, dustCount),
		radarTrails:      make(map[int][]RadarTrailPoint),
		radarTrailTimers: make(map[int]float64),
		npcStates:        make(map[int]NPCState),
	}
	g.initFactions()

	// Create a few ships; index 0 is player, others are passive demo ships.
	g.ships = []Ship{
		{
			pos:      vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5},
			health:   100,
			isPlayer: true,
			faction:  "Union",
		},
		{
			pos:     vec2{float64(screenWidth)*0.5 + 120, float64(screenHeight)*0.5 - 60},
			angle:   math.Pi * 0.25,
			vel:     vec2{30, -10},
			health:  100,
			faction: "Raiders",
		},
		{
			pos:     vec2{float64(screenWidth)*0.5 - 160, float64(screenHeight)*0.5 + 90},
			angle:   -math.Pi * 0.5,
			vel:     vec2{-20, 25},
			health:  100,
			faction: "Raiders",
		},
		{
			pos:     vec2{float64(screenWidth)*0.5 + 220, float64(screenHeight)*0.5 + 40},
			angle:   math.Pi * 0.15,
			vel:     vec2{15, 5},
			health:  100,
			faction: "Traders", // Allied with the player to support friendly ships later.
		},
	}
	g.playerIndex = 0

	// Seed dust in a square around the player so rotated views stay filled.
	initialSpan := math.Hypot(float64(screenWidth), float64(screenHeight)) * dustSpanMultiplier
	halfSpan := initialSpan * 0.5
	for i := range g.dust {
		g.dust[i] = dust{
			pos: vec2{
				x: g.ships[g.playerIndex].pos.x + rand.Float64()*initialSpan - halfSpan,
				y: g.ships[g.playerIndex].pos.y + rand.Float64()*initialSpan - halfSpan,
			},
			speed:  0.5 + rand.Float64()*1.0, // Speed multiplier from 0.5x to 1.5x
			radius: 1,
		}
	}

	return g
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0

	g.handleInput()

	player := &g.ships[g.playerIndex]

	// Update all ships using unified physics system
	for i := range g.ships {
		ship := &g.ships[i]
		var input ShipInput

		if ship.isPlayer {
			// Player: read keyboard input
			input = getPlayerInput()
		} else {
			// NPC: generate input from AI state machine
			input = g.updateNPCStateMachine(ship, player, dt)
		}

		// Apply physics using unified system
		g.updatePhysics(ship, input, dt)
	}

	g.updateDust(dt, player)

	// Update radar trails
	g.updateRadarTrails(dt, player)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(colorBackground)

	player := &g.ships[g.playerIndex]
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}

	// Draw dust (already positioned relative to ship movement)
	for _, d := range g.dust {
		offset := vec2{d.pos.x - player.pos.x, d.pos.y - player.pos.y}
		rot := rotatePoint(offset, -player.angle)
		drawCircle(screen, screenCenter.x+rot.x, screenCenter.y+rot.y, d.radius, colorDust)
	}

	for i := range g.ships {
		ship := &g.ships[i]
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
		g.drawShip(screen, ship, shipScreenX, shipScreenY, renderAngle, velRender)
	}

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
