package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	screenWidth              = 900
	screenHeight             = 600
	angularAccel             = math.Pi * 6 // radians per second^2
	angularDampingAccel      = math.Pi * 8 // radians per second^2 (for S key)
	maxAngularSpeed          = math.Pi * 4 // maximum angular speed (radians per second)
	thrustAccel              = 230.0       // pixels per second^2
	starCount                = 120
	starBaseSpeed            = 20.0
	retroAlignTolerance      = 20 * math.Pi / 180 // radians
	retroVelocityStopEpsilon = 5.0                // px/s, consider ship stopped
	retroMinSpeedForTurn     = 1.0                // px/s, minimum speed to compute heading
	radarRadius              = 70.0
	radarRange               = 520.0
	radarMargin              = 14.0
	indicatorMargin          = 18.0
	indicatorArrowLen        = 18.0
)

type vec2 struct {
	x float64
	y float64
}

// Ship represents a single spacecraft in the world.
type Ship struct {
	pos                 vec2
	vel                 vec2
	angle               float64
	angularVel          float64
	health              float64
	faction             string
	thrustThisFrame     bool
	turningThisFrame    bool
	turnDirection       float64 // -1 for left, 1 for right, 0 for none
	dampingAngularSpeed bool    // true when S key is pressed to dampen angular speed
	retrogradeMode      bool    // true when performing retrograde burn maneuver
	retrogradeTurnDir   float64 // chosen turn direction for retrograde (-1 or 1)
	isPlayer            bool
}

type star struct {
	pos    vec2
	speed  float64
	radius float64
}

// Game holds the minimal state required for a simple arcade-feel spaceship demo.
type Game struct {
	ships         []Ship
	playerIndex   int
	stars         []star
	factionColors map[string]color.NRGBA
	alliances     map[string]map[string]bool
}

func newGame() *Game {
	rand.Seed(time.Now().UnixNano())

	g := &Game{
		stars: make([]star, starCount),
	}
	g.initFactions()

	// Create a few ships; index 0 is player, others are passive demo ships.
	g.ships = []Ship{
		{
			pos:      vec2{screenWidth * 0.5, screenHeight * 0.5},
			health:   100,
			isPlayer: true,
			faction:  "Union",
		},
		{
			pos:     vec2{screenWidth*0.5 + 120, screenHeight*0.5 - 60},
			angle:   math.Pi * 0.25,
			vel:     vec2{30, -10},
			health:  100,
			faction: "Raiders",
		},
		{
			pos:     vec2{screenWidth*0.5 - 160, screenHeight*0.5 + 90},
			angle:   -math.Pi * 0.5,
			vel:     vec2{-20, 25},
			health:  100,
			faction: "Raiders",
		},
		{
			pos:     vec2{screenWidth*0.5 + 220, screenHeight*0.5 + 40},
			angle:   math.Pi * 0.15,
			vel:     vec2{15, 5},
			health:  100,
			faction: "Traders", // Allied with the player to support friendly ships later.
		},
	}
	g.playerIndex = 0

	for i := range g.stars {
		g.stars[i] = star{
			pos: vec2{
				x: rand.Float64() * screenWidth,
				y: rand.Float64() * screenHeight,
			},
			speed:  starBaseSpeed + rand.Float64()*starBaseSpeed,
			radius: 1 + rand.Float64()*1.5,
		}
	}

	return g
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0
	player := &g.ships[g.playerIndex]
	g.updatePhysics(player, dt)
	g.updateStars(dt, player.vel)

	// Move non-player ships with their own velocities (no controls for now).
	for i := range g.ships {
		if i == g.playerIndex {
			continue
		}
		g.ships[i].pos.x += g.ships[i].vel.x * dt
		g.ships[i].pos.y += g.ships[i].vel.y * dt
	}
	return nil
}

func (g *Game) updateStars(dt float64, cameraVel vec2) {
	// Move stars relative to ship velocity (opposite direction for parallax effect)
	for i := range g.stars {
		// Stars move opposite to ship movement
		g.stars[i].pos.x -= cameraVel.x * dt
		g.stars[i].pos.y -= cameraVel.y * dt

		// Wrap stars around screen bounds
		if g.stars[i].pos.x < 0 {
			g.stars[i].pos.x += screenWidth
		}
		if g.stars[i].pos.x > screenWidth {
			g.stars[i].pos.x -= screenWidth
		}
		if g.stars[i].pos.y < 0 {
			g.stars[i].pos.y += screenHeight
		}
		if g.stars[i].pos.y > screenHeight {
			g.stars[i].pos.y -= screenHeight
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 3, G: 5, B: 16, A: 255})

	// Draw stars (already positioned relative to ship movement)
	for _, s := range g.stars {
		drawCircle(screen, s.pos.x, s.pos.y, s.radius, color.NRGBA{R: 200, G: 200, B: 255, A: 255})
	}

	player := &g.ships[g.playerIndex]
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}
	for i := range g.ships {
		ship := &g.ships[i]
		// Position relative to player so camera is centered on player ship.
		offsetX := ship.pos.x - player.pos.x
		offsetY := ship.pos.y - player.pos.y
		shipScreenX := screenCenter.x + offsetX
		shipScreenY := screenCenter.y + offsetY
		g.drawShip(screen, ship, shipScreenX, shipScreenY)
	}

	g.drawOffscreenIndicators(screen, player)

	retroStatus := ""
	if player.retrogradeMode {
		speed := math.Hypot(player.vel.x, player.vel.y)
		targetAngle := math.Atan2(-player.vel.x, player.vel.y)
		angleDiff := math.Abs(normalizeAngle(targetAngle-player.angle)) * 180 / math.Pi
		if angleDiff > 20 {
			retroStatus = fmt.Sprintf(" | RETROGRADE: TURNING (%.0f° off)", angleDiff)
		} else {
			retroStatus = fmt.Sprintf(" | RETROGRADE: BURNING (speed: %.1f)", speed)
		}
	}
	hud := fmt.Sprintf("S: Retrograde Burn | Speed: %0.1f | Angular: %0.2f rad/s%s",
		math.Hypot(player.vel.x, player.vel.y), player.angularVel, retroStatus)
	ebitenutil.DebugPrint(screen, hud)

	g.drawRadar(screen, player)
}

func (g *Game) drawShip(screen *ebiten.Image, ship *Ship, shipCenterX, shipCenterY float64) {
	// Draw player brighter; others dimmer.
	var shipColor color.Color = color.White
	if !ship.isPlayer {
		shipColor = g.colorForFaction(ship.faction)
	}

	// Triangle points for the ship in local space (nose up)
	nose := rotatePoint(vec2{0, -18}, ship.angle)
	left := rotatePoint(vec2{-12, 12}, ship.angle)
	right := rotatePoint(vec2{12, 12}, ship.angle)

	nose.x += shipCenterX
	nose.y += shipCenterY
	left.x += shipCenterX
	left.y += shipCenterY
	right.x += shipCenterX
	right.y += shipCenterY

	ebitenutil.DrawLine(screen, nose.x, nose.y, left.x, left.y, shipColor)
	ebitenutil.DrawLine(screen, left.x, left.y, right.x, right.y, shipColor)
	ebitenutil.DrawLine(screen, right.x, right.y, nose.x, nose.y, shipColor)

	// Draw green velocity vector on top of ship
	velocityScale := 0.1 // Scale factor for visibility
	velEndX := shipCenterX + ship.vel.x*velocityScale
	velEndY := shipCenterY + ship.vel.y*velocityScale
	ebitenutil.DrawLine(screen, shipCenterX, shipCenterY, velEndX, velEndY, color.NRGBA{R: 0, G: 255, B: 0, A: 255})

	if ship.thrustThisFrame {
		// Position flame at the back center of the ship (midpoint of left and right back points)
		flameAnchor := rotatePoint(vec2{0, 12}, ship.angle)
		flameAnchor.x += shipCenterX
		flameAnchor.y += shipCenterY

		// Flame extends backward from the ship (opposite direction of forward movement)
		// The back is at y=12, so we extend further back (positive y in local space)
		flameLength := 28 + rand.Float64()*8
		flameDir := rotatePoint(vec2{0, 12 + flameLength}, ship.angle)
		flameDir.x += shipCenterX
		flameDir.y += shipCenterY

		flameColor := color.NRGBA{R: 255, G: 150 + uint8(rand.Intn(100)), B: 0, A: 255}
		ebitenutil.DrawLine(screen, flameAnchor.x, flameAnchor.y, flameDir.x, flameDir.y, flameColor)
	}

	// Draw sideways flames when actively turning (only when input is pressed)
	if ship.turningThisFrame {
		if ship.turnDirection > 0 {
			// Turning right - show flame on right side
			g.fireThruster(screen, ship, true, shipCenterX, shipCenterY) // right
		} else {
			// Turning left - show flame on left side
			g.fireThruster(screen, ship, false, shipCenterX, shipCenterY) // left
		}
	}

	// Automatically fire rotation cancellation thruster when no turn input but still rotating
	if !ship.turningThisFrame && math.Abs(ship.angularVel) > 0.1 {
		// Fire thruster on the side that opposes current rotation
		if ship.angularVel > 0 {
			// Rotating right, fire left thruster to counter
			g.fireThruster(screen, ship, false, shipCenterX, shipCenterY) // left
		} else {
			// Rotating left, fire right thruster to counter
			g.fireThruster(screen, ship, true, shipCenterX, shipCenterY) // right
		}
	}

	// Draw angular damping thruster when S is pressed (fires on side that opposes rotation)
	// S key provides stronger/faster damping
	if ship.dampingAngularSpeed && math.Abs(ship.angularVel) > 0.1 {
		// Fire thruster on the side that opposes current rotation
		if ship.angularVel > 0 {
			// Rotating right, fire left thruster to counter
			g.fireThruster(screen, ship, false, shipCenterX, shipCenterY) // left
		} else {
			// Rotating left, fire right thruster to counter
			g.fireThruster(screen, ship, true, shipCenterX, shipCenterY) // right
		}
	}
}

// drawRadar renders a simple orientable radar in the top-right corner showing nearby enemies.
func (g *Game) drawRadar(screen *ebiten.Image, player *Ship) {
	center := vec2{
		x: float64(screenWidth) - radarMargin - radarRadius,
		y: radarMargin + radarRadius,
	}

	// Radar backdrop and crosshair
	drawCircle(screen, center.x, center.y, radarRadius+4, color.NRGBA{R: 10, G: 16, B: 32, A: 230})
	drawCircle(screen, center.x, center.y, radarRadius, color.NRGBA{R: 24, G: 48, B: 96, A: 255}) // outer ring
	ebitenutil.DrawLine(screen, center.x-radarRadius, center.y, center.x+radarRadius, center.y, color.NRGBA{R: 50, G: 80, B: 120, A: 255})
	ebitenutil.DrawLine(screen, center.x, center.y-radarRadius, center.x, center.y+radarRadius, color.NRGBA{R: 50, G: 80, B: 120, A: 255})

	// Player heading marker (shows facing direction relative to fixed-world radar)
	headingLen := radarRadius - 8
	headX := center.x + math.Sin(player.angle)*headingLen
	headY := center.y - math.Cos(player.angle)*headingLen
	ebitenutil.DrawLine(screen, center.x, center.y, headX, headY, color.NRGBA{R: 120, G: 210, B: 255, A: 255})
	drawCircle(screen, center.x, center.y, 2, color.NRGBA{R: 180, G: 255, B: 200, A: 255})

	// Fixed-world radar (no rotation). +X right, +Y down.
	scale := radarRadius / radarRange

	for i := range g.ships {
		if i == g.playerIndex {
			continue
		}
		enemy := &g.ships[i]

		dx := enemy.pos.x - player.pos.x
		dy := enemy.pos.y - player.pos.y

		dist := math.Hypot(dx, dy)

		rx := dx * scale
		ry := dy * scale

		blipColor := g.colorForFaction(enemy.faction)
		isOffRadar := dist > radarRange
		if isOffRadar {
			// Place on the edge of the radar circle and show distance
			dirX := dx / dist
			dirY := dy / dist
			maxR := radarRadius - 5
			rx = dirX * maxR
			ry = dirY * maxR

			label := fmt.Sprintf("%.0f", dist)
			labelX := center.x + rx + dirX*10
			labelY := center.y + ry + dirY*10
			minX := center.x - radarRadius + 6
			maxX := center.x + radarRadius - 32
			minY := center.y - radarRadius + 6
			maxY := center.y + radarRadius - 12
			if labelX < minX {
				labelX = minX
			}
			if labelX > maxX {
				labelX = maxX
			}
			if labelY < minY {
				labelY = minY
			}
			if labelY > maxY {
				labelY = maxY
			}
			ebitenutil.DebugPrintAt(screen, label, int(labelX), int(labelY))
		} else {
			// Clamp to radar edge so distant targets sit on the rim
			if edgeDist := math.Hypot(rx, ry); edgeDist > radarRadius-4 {
				f := (radarRadius - 4) / edgeDist
				rx *= f
				ry *= f
			}
		}

		drawCircle(screen, center.x+rx, center.y+ry, 3, blipColor)
	}
}

// drawOffscreenIndicators draws edge-of-screen markers for enemies that are not visible.
func (g *Game) drawOffscreenIndicators(screen *ebiten.Image, player *Ship) {
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}
	minX := indicatorMargin
	maxX := float64(screenWidth) - indicatorMargin
	minY := indicatorMargin
	maxY := float64(screenHeight) - indicatorMargin

	type cornerStat struct {
		count   int
		minDist float64
		dir     vec2
		pos     vec2
		clr     color.Color
	}
	corners := map[string]*cornerStat{}

	drawIndicator := func(pos vec2, dir vec2, dist float64, count int, clr color.Color) {
		tipX := pos.x + dir.x*indicatorArrowLen*0.6
		tipY := pos.y + dir.y*indicatorArrowLen*0.6
		tailX := pos.x - dir.x*indicatorArrowLen*0.4
		tailY := pos.y - dir.y*indicatorArrowLen*0.4
		ebitenutil.DrawLine(screen, tailX, tailY, tipX, tipY, clr)

		wingAngle := math.Pi / 6
		sinA := math.Sin(wingAngle)
		cosA := math.Cos(wingAngle)
		leftWing := vec2{
			x: dir.x*cosA - dir.y*sinA,
			y: dir.x*sinA + dir.y*cosA,
		}
		rightWing := vec2{
			x: dir.x*cosA + dir.y*sinA,
			y: -dir.x*sinA + dir.y*cosA,
		}
		wingLen := indicatorArrowLen * 0.5
		ebitenutil.DrawLine(screen, tipX, tipY, tipX-leftWing.x*wingLen, tipY-leftWing.y*wingLen, clr)
		ebitenutil.DrawLine(screen, tipX, tipY, tipX-rightWing.x*wingLen, tipY-rightWing.y*wingLen, clr)

		label := fmt.Sprintf("%.0f", dist)
		if count > 1 {
			label = fmt.Sprintf("%.0f (x%d)", dist, count)
		}
		labelX := pos.x + 8
		labelY := pos.y - 8
		maxLabelX := float64(screenWidth) - 64 // leave more room for multiplier text
		if labelX > maxLabelX {
			labelX = maxLabelX
		}
		if labelX < 4 {
			labelX = 4
		}
		if labelY < 4 {
			labelY = 4
		}
		if labelY > float64(screenHeight)-12 {
			labelY = float64(screenHeight) - 12
		}
		ebitenutil.DebugPrintAt(screen, label, int(labelX), int(labelY))
	}

	for i := range g.ships {
		if i == g.playerIndex {
			continue
		}
		enemy := &g.ships[i]
		indicatorColor := g.colorForFaction(enemy.faction)

		dx := enemy.pos.x - player.pos.x
		dy := enemy.pos.y - player.pos.y
		dist := math.Hypot(dx, dy)
		if dist < 1 {
			continue
		}

		screenX := screenCenter.x + dx
		screenY := screenCenter.y + dy

		// If on-screen, skip indicator.
		if screenX >= 0 && screenX <= float64(screenWidth) && screenY >= 0 && screenY <= float64(screenHeight) {
			continue
		}

		// Clamp to edge with margin.
		clampedX := math.Min(math.Max(screenX, minX), maxX)
		clampedY := math.Min(math.Max(screenY, minY), maxY)

		dirX := dx / dist
		dirY := dy / dist

		isCorner := (clampedX == minX || clampedX == maxX) && (clampedY == minY || clampedY == maxY)
		if isCorner {
			key := fmt.Sprintf("%t-%t", clampedX == minX, clampedY == minY) // left/right - top/bottom
			if stat, ok := corners[key]; ok {
				stat.count++
				if dist < stat.minDist {
					stat.minDist = dist
					stat.dir = vec2{dirX, dirY}
					stat.pos = vec2{clampedX, clampedY}
					stat.clr = indicatorColor
				}
			} else {
				corners[key] = &cornerStat{
					count:   1,
					minDist: dist,
					dir:     vec2{dirX, dirY},
					pos:     vec2{clampedX, clampedY},
					clr:     indicatorColor,
				}
			}
			continue
		}

		drawIndicator(vec2{clampedX, clampedY}, vec2{dirX, dirY}, dist, 1, indicatorColor)
	}

	for _, stat := range corners {
		drawIndicator(stat.pos, stat.dir, stat.minDist, stat.count, stat.clr)
	}
}

func (g *Game) fireThruster(screen *ebiten.Image, ship *Ship, right bool, centerX, centerY float64) {
	// right: true for right side, false for left side
	sideOffset := -10.0 // left side
	if right {
		sideOffset = 10.0 // right side
	}

	sideFlameLength := 15 + rand.Float64()*5
	sideFlameColor := color.NRGBA{R: 255, G: 120 + uint8(rand.Intn(80)), B: 0, A: 255}

	// Position flame anchor on the side of the ship, near the back
	flameAnchor := rotatePoint(vec2{sideOffset, 8}, ship.angle)
	flameAnchor.x += centerX
	flameAnchor.y += centerY

	// Outward direction: (1, 0) for right side, (-1, 0) for left side in local space
	outwardDirX := -1.0 // left
	if right {
		outwardDirX = 1.0 // right
	}
	outwardDir := rotatePoint(vec2{outwardDirX, 0}, ship.angle)

	flameDir := vec2{
		x: flameAnchor.x + outwardDir.x*sideFlameLength,
		y: flameAnchor.y + outwardDir.y*sideFlameLength,
	}

	ebitenutil.DrawLine(screen, flameAnchor.x, flameAnchor.y, flameDir.x, flameDir.y, sideFlameColor)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func rotatePoint(p vec2, angle float64) vec2 {
	sinA := math.Sin(angle)
	cosA := math.Cos(angle)
	return vec2{
		x: p.x*cosA - p.y*sinA,
		y: p.x*sinA + p.y*cosA,
	}
}

func drawCircle(dst *ebiten.Image, cx, cy, radius float64, clr color.Color) {
	// Very cheap filled circle for the simple star field.
	steps := int(radius*4 + 4)
	for i := 0; i < steps; i++ {
		angle := float64(i) / float64(steps) * 2 * math.Pi
		x := cx + math.Cos(angle)*radius
		y := cy + math.Sin(angle)*radius
		dst.Set(int(x), int(y), clr)
	}
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Pocket Rocket - Ebiten Demo")
	ebiten.SetTPS(60)

	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
