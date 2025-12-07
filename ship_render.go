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
	// Rocks
	if g.isRock(ship) {
		if g.rockImage != nil {
			op := &ebiten.DrawImageOptions{}
			w, h := g.rockImage.Size()
			// Scale to match rock radius (slightly larger for visual cover)
			maxDim := math.Max(float64(w), float64(h))
			scale := (rockRadius * 2.0) / maxDim

			op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
			op.GeoM.Scale(scale, scale)
			// Rocks rotate based on their unique ID or just random context if we tracked it,
			// but for now, we can use their movement angle or just 0 since they don't have angularVel.
			// Let's just use 0 or a fixed rotation if we had it.
			// Actually, let's rotate it by time slightly to make it look dynamic?
			// But rocks are static in orientation in the physics currently.
			// Let's rotate by its position pseudo-randomly to vary look
			op.GeoM.Rotate(ship.pos.x*0.01 + ship.pos.y*0.01)
			op.GeoM.Translate(shipCenterX, shipCenterY)

			// Tint if on collision course
			if g.isOnCollisionCourse(player, ship, collisionCourseLookAhead) {
				op.ColorM.Scale(1, 0.5, 0.5, 1) // Red tint
				// Glow
				glowColor := color.NRGBA{R: 255, G: 100, B: 100, A: 64}
				drawCircle(screen, shipCenterX, shipCenterY, rockRadius+6, glowColor)
			}

			screen.DrawImage(g.rockImage, op)
		} else {
			// Fallback rendering
			rockColor := g.colorForFaction(ship.faction)
			if g.isOnCollisionCourse(player, ship, collisionCourseLookAhead) {
				glowColor := color.NRGBA{R: 255, G: 100, B: 100, A: 128}
				drawCircle(screen, shipCenterX, shipCenterY, rockRadius+4, glowColor)
				drawCircle(screen, shipCenterX, shipCenterY, rockRadius, colorRockCollision)
			} else {
				drawCircle(screen, shipCenterX, shipCenterY, rockRadius, rockColor)
			}
		}
		return
	}

	// Draw player brighter; others dimmer.
	var shipColor color.Color = color.White
	if !ship.isPlayer {
		shipColor = g.colorForFaction(ship.faction)
	}

	// Determine ship image
	var img *ebiten.Image
	if ship.isPlayer {
		img = g.playerImage
	} else {
		img = g.enemyImage
	}

	if img != nil {
		op := &ebiten.DrawImageOptions{}
		w, h := img.Size()
		// Scale based on max dimension to ensure proper sizing
		maxDim := math.Max(float64(w), float64(h))
		scale := (shipCollisionRadius * 2.5) / maxDim
		op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
		op.GeoM.Scale(scale, scale)
		op.GeoM.Rotate(renderAngle)
		op.GeoM.Translate(shipCenterX, shipCenterY)

		// Apply faction color tinting (player stays bright, others get faction colors)
		if !ship.isPlayer {
			factionColor := g.colorForFaction(ship.faction)
			// Convert NRGBA to RGB multipliers (0-1 range)
			op.ColorM.Scale(
				float64(factionColor.R)/255.0,
				float64(factionColor.G)/255.0,
				float64(factionColor.B)/255.0,
				1.0,
			)
		}

		screen.DrawImage(img, op)
	} else {
		// Fallback vector rendering
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

		// Draw turret points only in fallback
		for _, turretLocal := range ship.turretPoints {
			turretRotated := rotatePoint(turretLocal, renderAngle)
			turretX := shipCenterX + turretRotated.x
			turretY := shipCenterY + turretRotated.y
			turretColor := color.NRGBA{R: 200, G: 200, B: 200, A: 255}
			drawCircle(screen, turretX, turretY, turretSize, turretColor)
		}
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
