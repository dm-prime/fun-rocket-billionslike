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
	screenWidth        = 900
	screenHeight       = 600
	angularAccel       = math.Pi * 6  // radians per second^2
	angularDampingAccel = math.Pi * 8  // radians per second^2 (for S key)
	maxAngularSpeed    = math.Pi * 4   // maximum angular speed (radians per second)
	thrustAccel        = 230.0         // pixels per second^2
	starCount          = 120
	starBaseSpeed      = 20.0
)

type vec2 struct {
	x float64
	y float64
}

type star struct {
	pos    vec2
	speed  float64
	radius float64
}

// Game holds the minimal state required for a simple arcade-feel spaceship demo.
type Game struct {
	shipPos         vec2
	shipVel         vec2
	shipAngle       float64
	shipAngularVel  float64 // angular velocity in radians per second
	health          float64
	stars           []star
	thrustThisFrame      bool
	turningThisFrame     bool
	turnDirection        float64 // -1 for left, 1 for right, 0 for none
	dampingAngularSpeed  bool    // true when S key is pressed to dampen angular speed
}

func newGame() *Game {
	rand.Seed(time.Now().UnixNano())

	g := &Game{
		shipPos: vec2{screenWidth * 0.5, screenHeight * 0.5},
		health:  100.0,
		stars:   make([]star, starCount),
	}

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
	g.thrustThisFrame = false
	g.turningThisFrame = false
	g.turnDirection = 0
	g.dampingAngularSpeed = false

	// Apply angular acceleration based on input
	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.shipAngularVel -= angularAccel * dt
		g.turningThisFrame = true
		g.turnDirection = -1
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.shipAngularVel += angularAccel * dt
		g.turningThisFrame = true
		g.turnDirection = 1
	}

	// Clamp angular velocity to max speed
	if g.shipAngularVel > maxAngularSpeed {
		g.shipAngularVel = maxAngularSpeed
	}
	if g.shipAngularVel < -maxAngularSpeed {
		g.shipAngularVel = -maxAngularSpeed
	}

	// Update ship angle based on angular velocity
	g.shipAngle += g.shipAngularVel * dt

	forwardX := math.Sin(g.shipAngle)
	forwardY := -math.Cos(g.shipAngle)

	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.shipVel.x += forwardX * thrustAccel * dt
		g.shipVel.y += forwardY * thrustAccel * dt
		g.thrustThisFrame = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		// Apply angular damping to reduce angular speed
		g.dampingAngularSpeed = true
		if g.shipAngularVel > 0 {
			g.shipAngularVel -= angularDampingAccel * dt
			if g.shipAngularVel < 0 {
				g.shipAngularVel = 0
			}
		} else if g.shipAngularVel < 0 {
			g.shipAngularVel += angularDampingAccel * dt
			if g.shipAngularVel > 0 {
				g.shipAngularVel = 0
			}
		}
	}

	g.shipPos.x += g.shipVel.x * dt
	g.shipPos.y += g.shipVel.y * dt

	if g.shipPos.x < 0 {
		g.shipPos.x += screenWidth
	}
	if g.shipPos.x > screenWidth {
		g.shipPos.x -= screenWidth
	}
	if g.shipPos.y < 0 {
		g.shipPos.y += screenHeight
	}
	if g.shipPos.y > screenHeight {
		g.shipPos.y -= screenHeight
	}

	g.updateStars(dt)
	return nil
}

func (g *Game) updateStars(dt float64) {
	// Give stars a slight parallax: faster movement when the ship speeds up.
	speedBoost := math.Hypot(g.shipVel.x, g.shipVel.y) * 0.05

	for i := range g.stars {
		g.stars[i].pos.y += (g.stars[i].speed + speedBoost) * dt
		if g.stars[i].pos.y > screenHeight {
			g.stars[i].pos.y = 0
			g.stars[i].pos.x = rand.Float64() * screenWidth
		}
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{R: 3, G: 5, B: 16, A: 255})

	for _, s := range g.stars {
		drawCircle(screen, s.pos.x, s.pos.y, s.radius, color.NRGBA{R: 200, G: 200, B: 255, A: 255})
	}

	g.drawShip(screen)

	hud := fmt.Sprintf("Arrow keys / WASD to steer | Speed: %0.1f | Angular Speed: %0.2f rad/s | Health: %0.0f", 
		math.Hypot(g.shipVel.x, g.shipVel.y), g.shipAngularVel, g.health)
	ebitenutil.DebugPrint(screen, hud)
}

func (g *Game) drawShip(screen *ebiten.Image) {
	// Triangle points for the ship in local space (nose up)
	nose := rotatePoint(vec2{0, -18}, g.shipAngle)
	left := rotatePoint(vec2{-12, 12}, g.shipAngle)
	right := rotatePoint(vec2{12, 12}, g.shipAngle)

	nose.x += g.shipPos.x
	nose.y += g.shipPos.y
	left.x += g.shipPos.x
	left.y += g.shipPos.y
	right.x += g.shipPos.x
	right.y += g.shipPos.y

	ebitenutil.DrawLine(screen, nose.x, nose.y, left.x, left.y, color.White)
	ebitenutil.DrawLine(screen, left.x, left.y, right.x, right.y, color.White)
	ebitenutil.DrawLine(screen, right.x, right.y, nose.x, nose.y, color.White)

	if g.thrustThisFrame {
		// Position flame at the back center of the ship (midpoint of left and right back points)
		flameAnchor := rotatePoint(vec2{0, 12}, g.shipAngle)
		flameAnchor.x += g.shipPos.x
		flameAnchor.y += g.shipPos.y

		// Flame extends backward from the ship (opposite direction of forward movement)
		// The back is at y=12, so we extend further back (positive y in local space)
		flameLength := 28 + rand.Float64()*8
		flameDir := rotatePoint(vec2{0, 12 + flameLength}, g.shipAngle)
		flameDir.x += g.shipPos.x
		flameDir.y += g.shipPos.y

		flameColor := color.NRGBA{R: 255, G: 150 + uint8(rand.Intn(100)), B: 0, A: 255}
		ebitenutil.DrawLine(screen, flameAnchor.x, flameAnchor.y, flameDir.x, flameDir.y, flameColor)
	}

	// Draw sideways flames when actively turning (only when input is pressed)
	if g.turningThisFrame {
		if g.turnDirection > 0 {
			// Turning right - show flame on right side
			g.fireThruster(screen, true)  // right
		} else {
			// Turning left - show flame on left side
			g.fireThruster(screen, false) // left
		}
	}

	// Draw angular damping thruster when S is pressed (fires on side that opposes rotation)
	if g.dampingAngularSpeed && math.Abs(g.shipAngularVel) > 0.1 {
		// Fire thruster on the side that opposes current rotation
		if g.shipAngularVel > 0 {
			// Rotating right, fire left thruster to counter
			g.fireThruster(screen, false) // left
		} else {
			// Rotating left, fire right thruster to counter
			g.fireThruster(screen, true) // right
		}
	}
}

func (g *Game) fireThruster(screen *ebiten.Image, right bool) {
	// right: true for right side, false for left side
	sideOffset := -10.0 // left side
	if right {
		sideOffset = 10.0 // right side
	}

	sideFlameLength := 15 + rand.Float64()*5
	sideFlameColor := color.NRGBA{R: 255, G: 120 + uint8(rand.Intn(80)), B: 0, A: 255}

	// Position flame anchor on the side of the ship, near the back
	flameAnchor := rotatePoint(vec2{sideOffset, 8}, g.shipAngle)
	flameAnchor.x += g.shipPos.x
	flameAnchor.y += g.shipPos.y

	// Outward direction: (1, 0) for right side, (-1, 0) for left side in local space
	outwardDirX := -1.0 // left
	if right {
		outwardDirX = 1.0 // right
	}
	outwardDir := rotatePoint(vec2{outwardDirX, 0}, g.shipAngle)
	
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
