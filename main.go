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
	screenWidth   = 900
	screenHeight  = 600
	rotationSpeed = math.Pi * 2 // radians per second
	thrustAccel   = 230.0       // pixels per second^2
	driftDamping  = 0.995
	starCount     = 120
	starBaseSpeed = 20.0
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
	stars           []star
	thrustThisFrame bool
}

func newGame() *Game {
	rand.Seed(time.Now().UnixNano())

	g := &Game{
		shipPos: vec2{screenWidth * 0.5, screenHeight * 0.5},
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

	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.shipAngle -= rotationSpeed * dt
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.shipAngle += rotationSpeed * dt
	}

	forwardX := math.Sin(g.shipAngle)
	forwardY := -math.Cos(g.shipAngle)

	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.shipVel.x += forwardX * thrustAccel * dt
		g.shipVel.y += forwardY * thrustAccel * dt
		g.thrustThisFrame = true
	}

	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		g.shipVel.x -= forwardX * thrustAccel * dt * 0.5
		g.shipVel.y -= forwardY * thrustAccel * dt * 0.5
	}

	g.shipVel.x *= driftDamping
	g.shipVel.y *= driftDamping

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

	hud := fmt.Sprintf("Arrow keys / WASD to steer | Speed: %0.1f", math.Hypot(g.shipVel.x, g.shipVel.y))
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
