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
	angularAccel             = math.Pi * 3 // radians per second^2
	angularDampingAccel      = math.Pi * 8 // radians per second^2 (for S key)
	maxAngularSpeed          = math.Pi * 4 // maximum angular speed (radians per second)
	thrustAccel              = 350.0       // pixels per second^2
	sideThrustAccel          = 77.0        // pixels per second^2 (side thruster acceleration)
	dustCount                = 70
	dustBaseSpeed            = 20.0
	retroAlignTolerance      = 20 * math.Pi / 180 // radians
	retroVelocityStopEpsilon = 5.0                // px/s, consider ship stopped
	retroMinSpeedForTurn     = 1.0                // px/s, minimum speed to compute heading
	retroBurnAlignWindow     = 8 * math.Pi / 180  // radians, must be within this to burn
	radarRadius              = 200.0
	radarRange               = 1520.0
	radarMargin              = 14.0
	indicatorMargin          = 18.0
	indicatorArrowLen        = 18.0
	radarTrailMaxAge         = 3.0 // seconds
	radarTrailUpdateInterval = 0.1 // seconds between trail points
	radarTrailMaxPoints      = 30  // maximum trail points per ship
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

type dust struct {
	pos    vec2
	speed  float64
	radius float64
}

// RadarTrailPoint represents a single point in a ship's radar trail
type RadarTrailPoint struct {
	pos vec2    // world coordinates
	age float64 // age in seconds
}

// Game holds the minimal state required for a simple arcade-feel spaceship demo.

// Game holds the minimal state required for a simple arcade-feel spaceship demo.
type Game struct {
	ships            []Ship
	playerIndex      int
	dust             []dust
	factionColors    map[string]color.NRGBA
	alliances        map[string]map[string]bool
	radarTrails      map[int][]RadarTrailPoint // ship index -> trail points
	radarTrailTimers map[int]float64           // ship index -> time since last trail point
}

func newGame() *Game {
	rand.Seed(time.Now().UnixNano())

	g := &Game{
		dust:             make([]dust, dustCount),
		radarTrails:      make(map[int][]RadarTrailPoint),
		radarTrailTimers: make(map[int]float64),
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

	// Seed dust in a square around the player so rotated views stay filled.
	initialSpan := math.Hypot(screenWidth, screenHeight) * 1.5
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
	player := &g.ships[g.playerIndex]
	g.updatePhysics(player, dt)
	g.updateDust(dt, player)

	// Update NPC AI to follow the player
	for i := range g.ships {
		if i == g.playerIndex {
			continue
		}
		g.updateNPC(&g.ships[i], player, dt)
	}

	// Update radar trails
	g.updateRadarTrails(dt, player)

	return nil
}

func (g *Game) updateDust(dt float64, player *Ship) {
	// Move dust relative to ship velocity (opposite direction for parallax effect)
	span := math.Hypot(screenWidth, screenHeight) * 1.5 // square torus sized by diagonal
	half := span * 0.5
	for i := range g.dust {
		// Dust moves opposite to ship movement, with individual speed variance
		speedMultiplier := g.dust[i].speed
		g.dust[i].pos.x -= player.vel.x * dt * speedMultiplier
		g.dust[i].pos.y -= player.vel.y * dt * speedMultiplier

		// Keep dust in a torus around the player so they don't depend on absolute origin.
		dx := g.dust[i].pos.x - player.pos.x
		dy := g.dust[i].pos.y - player.pos.y

		if dx < -half {
			g.dust[i].pos.x += span
		}
		if dx > half {
			g.dust[i].pos.x -= span
		}
		if dy < -half {
			g.dust[i].pos.y += span
		}
		if dy > half {
			g.dust[i].pos.y -= span
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 3, G: 5, B: 16, A: 255})

	player := &g.ships[g.playerIndex]
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}

	// Draw dust (already positioned relative to ship movement)
	for _, d := range g.dust {
		offset := vec2{d.pos.x - player.pos.x, d.pos.y - player.pos.y}
		rot := rotatePoint(offset, -player.angle)
		drawCircle(screen, screenCenter.x+rot.x, screenCenter.y+rot.y, d.radius, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
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

// updateRadarTrails updates the trail points for all ships on the radar
func (g *Game) updateRadarTrails(dt float64, player *Ship) {
	for i := range g.ships {
		ship := &g.ships[i]

		// Initialize timer if needed
		if _, exists := g.radarTrailTimers[i]; !exists {
			g.radarTrailTimers[i] = 0
		}

		// Age existing trail points
		trail := g.radarTrails[i]
		for j := range trail {
			trail[j].age += dt
		}

		// Remove old trail points
		newTrail := make([]RadarTrailPoint, 0, len(trail))
		for _, point := range trail {
			if point.age < radarTrailMaxAge {
				newTrail = append(newTrail, point)
			}
		}
		g.radarTrails[i] = newTrail

		// Add new trail point periodically
		g.radarTrailTimers[i] += dt
		if g.radarTrailTimers[i] >= radarTrailUpdateInterval {
			// Add new point with world coordinates
			newPoint := RadarTrailPoint{pos: ship.pos, age: 0}
			g.radarTrails[i] = append(g.radarTrails[i], newPoint)

			// Limit trail length
			if len(g.radarTrails[i]) > radarTrailMaxPoints {
				g.radarTrails[i] = g.radarTrails[i][1:]
			}

			g.radarTrailTimers[i] = 0
		}
	}
}

func (g *Game) drawShip(screen *ebiten.Image, ship *Ship, shipCenterX, shipCenterY float64, renderAngle float64, velRender vec2) {
	// Draw player brighter; others dimmer.
	var shipColor color.Color = color.White
	if !ship.isPlayer {
		shipColor = g.colorForFaction(ship.faction)
	}

	// Triangle points for the ship in local space (nose up)
	nose := rotatePoint(vec2{0, -18}, renderAngle)
	left := rotatePoint(vec2{-12, 12}, renderAngle)
	right := rotatePoint(vec2{12, 12}, renderAngle)

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
	velEndX := shipCenterX + velRender.x*velocityScale
	velEndY := shipCenterY + velRender.y*velocityScale
	ebitenutil.DrawLine(screen, shipCenterX, shipCenterY, velEndX, velEndY, color.NRGBA{R: 0, G: 255, B: 0, A: 255})

	if ship.thrustThisFrame {
		// Position flame at the back center of the ship (midpoint of left and right back points)
		flameAnchor := rotatePoint(vec2{0, 12}, renderAngle)
		flameAnchor.x += shipCenterX
		flameAnchor.y += shipCenterY

		// Flame extends backward from the ship (opposite direction of forward movement)
		// The back is at y=12, so we extend further back (positive y in local space)
		flameLength := 28 + rand.Float64()*8
		flameDir := rotatePoint(vec2{0, 12 + flameLength}, renderAngle)
		flameDir.x += shipCenterX
		flameDir.y += shipCenterY

		flameColor := color.NRGBA{R: 255, G: 150 + uint8(rand.Intn(100)), B: 0, A: 255}
		ebitenutil.DrawLine(screen, flameAnchor.x, flameAnchor.y, flameDir.x, flameDir.y, flameColor)
	}

	// Draw sideways flames when actively turning (only when input is pressed)
	if ship.turningThisFrame {
		if ship.turnDirection > 0 {
			// Turning right - show flame on right side
			g.fireThruster(screen, ship, true, shipCenterX, shipCenterY, renderAngle) // right
		} else {
			// Turning left - show flame on left side
			g.fireThruster(screen, ship, false, shipCenterX, shipCenterY, renderAngle) // left
		}
	}

	// Automatically fire rotation cancellation thruster when no turn input but still rotating
	if !ship.turningThisFrame && math.Abs(ship.angularVel) > 0.1 {
		// Fire thruster on the side that opposes current rotation
		if ship.angularVel > 0 {
			// Rotating right, fire left thruster to counter
			g.fireThruster(screen, ship, false, shipCenterX, shipCenterY, renderAngle) // left
		} else {
			// Rotating left, fire right thruster to counter
			g.fireThruster(screen, ship, true, shipCenterX, shipCenterY, renderAngle) // right
		}
	}

	// Draw angular damping thruster when S is pressed (fires on side that opposes rotation)
	// S key provides stronger/faster damping
	if ship.dampingAngularSpeed && math.Abs(ship.angularVel) > 0.1 {
		// Fire thruster on the side that opposes current rotation
		if ship.angularVel > 0 {
			// Rotating right, fire left thruster to counter
			g.fireThruster(screen, ship, false, shipCenterX, shipCenterY, renderAngle) // left
		} else {
			// Rotating left, fire right thruster to counter
			g.fireThruster(screen, ship, true, shipCenterX, shipCenterY, renderAngle) // right
		}
	}
}

// drawRadar renders a simple orientable radar centered on the player ship showing nearby enemies.
func (g *Game) drawRadar(screen *ebiten.Image, player *Ship) {
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}
	center := screenCenter

	// Radar backdrop
	drawCircle(screen, center.x, center.y, radarRadius+4, color.NRGBA{R: 10, G: 16, B: 32, A: 230})
	drawCircle(screen, center.x, center.y, radarRadius, color.NRGBA{R: 24, G: 48, B: 96, A: 255}) // outer ring

	// Player heading marker (always points up since radar rotates with player view)
	headingLen := radarRadius - 8
	headX := center.x
	headY := center.y - headingLen
	ebitenutil.DrawLine(screen, center.x, center.y, headX, headY, color.NRGBA{R: 120, G: 210, B: 255, A: 255})
	drawCircle(screen, center.x, center.y, 2, color.NRGBA{R: 180, G: 255, B: 200, A: 255})

	// Rotated radar (matches game rotation style). Rotate enemy positions relative to player angle.
	scale := radarRadius / radarRange

	// Draw player trail
	playerTrail := g.radarTrails[g.playerIndex]
	if len(playerTrail) > 1 {
		playerColor := color.NRGBA{R: 180, G: 255, B: 200, A: 255} // Player color
		for j := 0; j < len(playerTrail)-1; j++ {
			p1 := playerTrail[j]
			p2 := playerTrail[j+1]

			// Transform world coordinates to radar coordinates (relative to current player position)
			dx1 := p1.pos.x - player.pos.x
			dy1 := p1.pos.y - player.pos.y
			rotated1 := rotatePoint(vec2{dx1, dy1}, -player.angle)
			rx1 := rotated1.x * scale
			ry1 := rotated1.y * scale

			dx2 := p2.pos.x - player.pos.x
			dy2 := p2.pos.y - player.pos.y
			rotated2 := rotatePoint(vec2{dx2, dy2}, -player.angle)
			rx2 := rotated2.x * scale
			ry2 := rotated2.y * scale

			// Clamp to radar edge if needed
			if edgeDist1 := math.Hypot(rx1, ry1); edgeDist1 > radarRadius-4 {
				f := (radarRadius - 4) / edgeDist1
				rx1 *= f
				ry1 *= f
			}
			if edgeDist2 := math.Hypot(rx2, ry2); edgeDist2 > radarRadius-4 {
				f := (radarRadius - 4) / edgeDist2
				rx2 *= f
				ry2 *= f
			}

			// Calculate opacity based on age (fade from full to transparent)
			age := (p1.age + p2.age) / 2.0
			opacity := 1.0 - (age / radarTrailMaxAge)
			if opacity < 0 {
				opacity = 0
			}
			if opacity > 1 {
				opacity = 1
			}

			// Create faded color
			trailColor := color.NRGBA{
				R: playerColor.R,
				G: playerColor.G,
				B: playerColor.B,
				A: uint8(float64(playerColor.A) * opacity * 0.6), // Max 60% opacity for trails
			}

			// Draw trail segment
			ebitenutil.DrawLine(
				screen,
				center.x+rx1,
				center.y+ry1,
				center.x+rx2,
				center.y+ry2,
				trailColor,
			)
		}
	}

	for i := range g.ships {
		if i == g.playerIndex {
			continue
		}
		enemy := &g.ships[i]

		dx := enemy.pos.x - player.pos.x
		dy := enemy.pos.y - player.pos.y

		dist := math.Hypot(dx, dy)

		// Rotate the offset relative to player angle (same as ship rendering)
		rotated := rotatePoint(vec2{dx, dy}, -player.angle)
		rx := rotated.x * scale
		ry := rotated.y * scale

		blipColor := g.colorForFaction(enemy.faction)
		isOffRadar := dist > radarRange
		if isOffRadar {
			// Place on the edge of the radar circle and show distance
			dirX := rotated.x / dist
			dirY := rotated.y / dist
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

		// Draw fading trail
		trail := g.radarTrails[i]
		if len(trail) > 1 {
			baseColor := g.colorForFaction(enemy.faction)
			for j := 0; j < len(trail)-1; j++ {
				p1 := trail[j]
				p2 := trail[j+1]

				// Transform world coordinates to radar coordinates
				dx1 := p1.pos.x - player.pos.x
				dy1 := p1.pos.y - player.pos.y
				rotated1 := rotatePoint(vec2{dx1, dy1}, -player.angle)
				rx1 := rotated1.x * scale
				ry1 := rotated1.y * scale

				dx2 := p2.pos.x - player.pos.x
				dy2 := p2.pos.y - player.pos.y
				rotated2 := rotatePoint(vec2{dx2, dy2}, -player.angle)
				rx2 := rotated2.x * scale
				ry2 := rotated2.y * scale

				// Clamp to radar edge if needed
				if edgeDist1 := math.Hypot(rx1, ry1); edgeDist1 > radarRadius-4 {
					f := (radarRadius - 4) / edgeDist1
					rx1 *= f
					ry1 *= f
				}
				if edgeDist2 := math.Hypot(rx2, ry2); edgeDist2 > radarRadius-4 {
					f := (radarRadius - 4) / edgeDist2
					rx2 *= f
					ry2 *= f
				}

				// Calculate opacity based on age (fade from full to transparent)
				age := (p1.age + p2.age) / 2.0
				opacity := 1.0 - (age / radarTrailMaxAge)
				if opacity < 0 {
					opacity = 0
				}
				if opacity > 1 {
					opacity = 1
				}

				// Create faded color
				trailColor := color.NRGBA{
					R: baseColor.R,
					G: baseColor.G,
					B: baseColor.B,
					A: uint8(float64(baseColor.A) * opacity * 0.6), // Max 60% opacity for trails
				}

				// Draw trail segment
				ebitenutil.DrawLine(
					screen,
					center.x+rx1,
					center.y+ry1,
					center.x+rx2,
					center.y+ry2,
					trailColor,
				)
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

		// Rotate world around player so player stays upright.
		rot := rotatePoint(vec2{dx, dy}, -player.angle)
		screenX := screenCenter.x + rot.x
		screenY := screenCenter.y + rot.y

		// If on-screen, skip indicator.
		if screenX >= 0 && screenX <= float64(screenWidth) && screenY >= 0 && screenY <= float64(screenHeight) {
			continue
		}

		// Clamp to edge with margin.
		clampedX := math.Min(math.Max(screenX, minX), maxX)
		clampedY := math.Min(math.Max(screenY, minY), maxY)

		dirX := rot.x / math.Hypot(rot.x, rot.y)
		dirY := rot.y / math.Hypot(rot.x, rot.y)

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

func (g *Game) fireThruster(screen *ebiten.Image, ship *Ship, right bool, centerX, centerY float64, renderAngle float64) {
	// right: true for right side, false for left side
	sideOffset := -10.0 // left side
	if right {
		sideOffset = 10.0 // right side
	}

	sideFlameLength := 15 + rand.Float64()*5
	sideFlameColor := color.NRGBA{R: 255, G: 120 + uint8(rand.Intn(80)), B: 0, A: 255}

	// Position flame anchor on the side of the ship, near the back
	flameAnchor := rotatePoint(vec2{sideOffset, 8}, renderAngle)
	flameAnchor.x += centerX
	flameAnchor.y += centerY

	// Outward direction: (1, 0) for right side, (-1, 0) for left side in local space
	outwardDirX := -1.0 // left
	if right {
		outwardDirX = 1.0 // right
	}
	outwardDir := rotatePoint(vec2{outwardDirX, 0}, renderAngle)

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
	// Very cheap filled circle for the simple dust field.
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
