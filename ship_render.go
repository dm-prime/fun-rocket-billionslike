package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// drawShip draws a ship with its thrusters and velocity vector
func (g *Game) drawShip(screen *ebiten.Image, ship *Ship, shipCenterX, shipCenterY float64, renderAngle float64, velRender vec2, player *Ship) {
	// Rocks are drawn as circles, not triangles
	if g.isRock(ship) {
		rockColor := g.colorForFaction(ship.faction)
		
		// Check if rock is on collision course with player
		if g.isOnCollisionCourse(player, ship, collisionCourseLookAhead) {
			// Highlight rock on collision course with bright red and glow effect
			// Draw outer glow (larger, semi-transparent circle)
			glowColor := color.NRGBA{R: 255, G: 100, B: 100, A: 128}
			drawCircle(screen, shipCenterX, shipCenterY, rockRadius+4, glowColor)
			// Draw main rock in bright red
			drawCircle(screen, shipCenterX, shipCenterY, rockRadius, colorRockCollision)
		} else {
			// Normal rock color
			drawCircle(screen, shipCenterX, shipCenterY, rockRadius, rockColor)
		}
		// Rocks don't have velocity vectors or thrusters
		return
	}

	// Draw player brighter; others dimmer.
	var shipColor color.Color = color.White
	if !ship.isPlayer {
		shipColor = g.colorForFaction(ship.faction)
	}

	// Triangle points for the ship in local space (nose up)
	nose := rotatePoint(vec2{0, shipNoseOffsetY}, renderAngle)
	left := rotatePoint(vec2{shipLeftOffsetX, shipLeftOffsetY}, renderAngle)
	right := rotatePoint(vec2{shipRightOffsetX, shipRightOffsetY}, renderAngle)

	nose.x += shipCenterX
	nose.y += shipCenterY
	left.x += shipCenterX
	left.y += shipCenterY
	right.x += shipCenterX
	right.y += shipCenterY

	ebitenutil.DrawLine(screen, nose.x, nose.y, left.x, left.y, shipColor)
	ebitenutil.DrawLine(screen, left.x, left.y, right.x, right.y, shipColor)
	ebitenutil.DrawLine(screen, right.x, right.y, nose.x, nose.y, shipColor)

	// Draw turret points
	for _, turretLocal := range ship.turretPoints {
		turretRotated := rotatePoint(turretLocal, renderAngle)
		turretX := shipCenterX + turretRotated.x
		turretY := shipCenterY + turretRotated.y
		// Draw turret as a small circle
		turretColor := color.NRGBA{R: 200, G: 200, B: 200, A: 255}
		drawCircle(screen, turretX, turretY, turretSize, turretColor)
	}

	// Draw green velocity vector for all ships (predictive trail is now in radar)
	velEndX := shipCenterX + velRender.x*velocityVectorScale
	velEndY := shipCenterY + velRender.y*velocityVectorScale
	ebitenutil.DrawLine(screen, shipCenterX, shipCenterY, velEndX, velEndY, colorVelocityVector)

	if ship.thrustThisFrame {
		// Position flame at the back center of the ship (midpoint of left and right back points)
		flameAnchor := rotatePoint(vec2{0, shipBackOffsetY}, renderAngle)
		flameAnchor.x += shipCenterX
		flameAnchor.y += shipCenterY

		// Flame extends backward from the ship (opposite direction of forward movement)
		flameLength := flameBaseLength + rand.Float64()*flameVarLength
		flameDir := rotatePoint(vec2{0, shipBackOffsetY + flameLength}, renderAngle)
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

// fireThruster draws a side thruster flame effect
func (g *Game) fireThruster(screen *ebiten.Image, ship *Ship, right bool, centerX, centerY float64, renderAngle float64) {
	// right: true for right side, false for left side
	sideOffset := -sideThrusterX // left side
	if right {
		sideOffset = sideThrusterX // right side
	}

	sideFlameLength := sideFlameBaseLen + rand.Float64()*sideFlameVarLen
	sideFlameColor := color.NRGBA{R: 255, G: 120 + uint8(rand.Intn(80)), B: 0, A: 255}

	// Position flame anchor on the side of the ship, near the back
	flameAnchor := rotatePoint(vec2{sideOffset, shipBackOffsetY}, renderAngle)
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
