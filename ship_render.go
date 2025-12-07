package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// drawShip draws a ship with its thrusters and velocity vector
func (g *Game) drawShip(screen *ebiten.Image, ship *Ship, shipCenterX, shipCenterY float64, renderAngle float64, velRender vec2, player *Ship) {

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

	// Particle systems now handle all thrust and turning effects
}

// drawRock draws a rock with collision highlighting if on collision course
func (g *Game) drawRock(screen *ebiten.Image, rock *Rock, rockCenterX, rockCenterY float64, player *Ship) {
	rockColor := g.colorForFaction("Rocks")

	// Check if rock is on collision course with player
	if g.isOnCollisionCourse(player, rock, collisionCourseLookAhead) {
		// Highlight rock on collision course with bright red and glow effect
		// Draw outer glow (larger, semi-transparent circle)
		glowColor := color.NRGBA{R: 255, G: 100, B: 100, A: 128}
		drawCircle(screen, rockCenterX, rockCenterY, rockRadius+4, glowColor)
		// Draw main rock in bright red
		drawCircle(screen, rockCenterX, rockCenterY, rockRadius, colorRockCollision)
	} else {
		// Normal rock color
		drawCircle(screen, rockCenterX, rockCenterY, rockRadius, rockColor)
	}
}
