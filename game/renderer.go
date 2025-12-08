package game

import (
	"bytes"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/goregular"
)

// Camera represents the viewport into the world
type Camera struct {
	X, Y     float64 // Camera position in world coordinates
	Zoom     float64 // Zoom level
	Width    float64 // Viewport width
	Height   float64 // Viewport height
	Rotation float64 // Camera rotation (not used in top-down)
}

// NewCamera creates a new camera
func NewCamera(width, height float64) *Camera {
	return &Camera{
		X:      0,
		Y:      0,
		Zoom:   1.0,
		Width:  width,
		Height: height,
	}
}

// WorldToScreen converts world coordinates to screen coordinates
func (c *Camera) WorldToScreen(wx, wy float64) (float64, float64) {
	// Translate by camera position
	sx := wx - c.X
	sy := wy - c.Y

	// Apply zoom
	sx *= c.Zoom
	sy *= c.Zoom

	// Translate to screen center
	sx += c.Width / 2
	sy += c.Height / 2

	return sx, sy
}

// ScreenToWorld converts screen coordinates to world coordinates
func (c *Camera) ScreenToWorld(sx, sy float64) (float64, float64) {
	// Translate from screen center
	wx := sx - c.Width/2
	wy := sy - c.Height/2

	// Apply inverse zoom
	wx /= c.Zoom
	wy /= c.Zoom

	// Translate by camera position
	wx += c.X
	wy += c.Y

	return wx, wy
}

// GetVisibleCells returns the cells visible in the camera viewport
func (c *Camera) GetVisibleCells(world *World) []*Cell {
	cells := make([]*Cell, 0, 100)

	// Get world bounds of viewport
	minX, minY := c.ScreenToWorld(0, 0)
	maxX, maxY := c.ScreenToWorld(c.Width, c.Height)

	// Expand bounds by cell size to include partially visible cells
	minX -= world.Config.CellSize
	minY -= world.Config.CellSize
	maxX += world.Config.CellSize
	maxY += world.Config.CellSize

	// Get cell range
	minCellX, minCellY := world.WorldToCell(minX, minY)
	maxCellX, maxCellY := world.WorldToCell(maxX, maxY)

	// Collect visible cells
	for x := minCellX; x <= maxCellX; x++ {
		for y := minCellY; y <= maxCellY; y++ {
			cell := world.GetCell(x, y)
			if cell != nil && cell.Count > 0 {
				cells = append(cells, cell)
			}
		}
	}

	return cells
}

// Renderer handles rendering of game entities
type Renderer struct {
	camera     *Camera
	faceSource *text.GoTextFaceSource
}

// NewRenderer creates a new renderer
func NewRenderer(camera *Camera) *Renderer {
	faceSource, _ := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	return &Renderer{
		camera:     camera,
		faceSource: faceSource,
	}
}

// Render renders all visible entities
func (r *Renderer) Render(screen *ebiten.Image, world *World, player *Entity, score int, fps float64) {
	// Get visible cells
	visibleCells := r.camera.GetVisibleCells(world)

	// Count visible entities to optimize rendering
	visibleEntityCount := 0
	for _, cell := range visibleCells {
		visibleEntityCount += cell.Count
	}
	
	// Skip expensive aim line rendering if there are too many entities
	drawAimLines := visibleEntityCount < 50

	// Render entities in visible cells
	for _, cell := range visibleCells {
		for _, entity := range cell.GetActiveEntities() {
			if entity.Health <= 0 {
				continue
			}
			r.renderEntityWithAim(screen, entity, player, drawAimLines)
		}
	}

	// Render UI (score, FPS, and restart message)
	r.RenderUI(screen, player, score, fps)
}

// RenderEntity renders a single entity
func (r *Renderer) RenderEntity(screen *ebiten.Image, entity *Entity, player *Entity) {
	r.renderEntityWithAim(screen, entity, player, true)
}

// renderEntityWithAim renders a single entity, with optional aim line rendering
func (r *Renderer) renderEntityWithAim(screen *ebiten.Image, entity *Entity, player *Entity, drawAimLines bool) {
	// Convert world coordinates to screen coordinates
	sx, sy := r.camera.WorldToScreen(entity.X, entity.Y)

	// Skip if outside screen bounds (with margin)
	margin := 100.0
	if sx < -margin || sx > r.camera.Width+margin ||
		sy < -margin || sy > r.camera.Height+margin {
		return
	}

	// Determine color based on ship type
	var factionConfig = GetFactionConfig(entity.Faction)
	var clr = factionConfig.Color
	if entity.Type == EntityTypeProjectile {
		// Color bullets by owner's ship type
		if entity.Owner != nil {
			clr = factionConfig.Color
		} else {
			clr = color.RGBA{255, 255, 0, 255} // Yellow fallback if no owner
		}
	} else {
		if entity.ShipType == ShipTypeHomingSuicide {
			clr = factionConfig.Color
		} else {
			clr = factionConfig.Color
		}
	}

	// Draw entity based on ship shape
	radius := entity.Radius * r.camera.Zoom
	if radius < 1 {
		radius = 1
	}

	// Get ship config for shape (cache it since we use it multiple times)
	var shipConfig ShipTypeConfig
	if entity.Type != EntityTypeProjectile {
		shipConfig = GetShipTypeConfig(entity.ShipType)
	} else {
		shipConfig = ShipTypeConfig{Shape: ShipShapeCircle}
	}

	// Draw entity based on shape
	switch shipConfig.Shape {
	case ShipShapeCircle:
		vector.DrawFilledCircle(screen, float32(sx), float32(sy), float32(radius), clr, true)
	case ShipShapeTriangle:
		r.drawTriangle(screen, sx, sy, radius, entity.Rotation, clr, entity.ShipType)
	case ShipShapeSquare:
		r.drawSquare(screen, sx, sy, radius, entity.Rotation, clr)
	case ShipShapeDiamond:
		r.drawDiamond(screen, sx, sy, radius, entity.Rotation, clr)
	default:
		vector.DrawFilledCircle(screen, float32(sx), float32(sy), float32(radius), clr, true)
	}

	// Draw direction indicator (small line)
	if entity.Type != EntityTypeProjectile {
		dirLength := radius * 1.5
		endX := sx + math.Cos(entity.Rotation)*dirLength
		endY := sy + math.Sin(entity.Rotation)*dirLength
		vector.StrokeLine(screen, float32(sx), float32(sy), float32(endX), float32(endY), 2, clr, true)
	}

	// Draw turret mount points (only for ships, not projectiles)
	// Reuse shipConfig we already fetched above
	if entity.Type != EntityTypeProjectile {
		for turretIndex, mount := range shipConfig.TurretMounts {
			// Only draw active turrets
			if !mount.Active {
				continue
			}

			// Calculate turret position relative to ship center
			// Rotate the offset by the ship's rotation
			cosRot := math.Cos(entity.Rotation)
			sinRot := math.Sin(entity.Rotation)

			// Transform mount offset from ship-local to world coordinates
			mountX := mount.OffsetX*cosRot - mount.OffsetY*sinRot
			mountY := mount.OffsetX*sinRot + mount.OffsetY*cosRot

			// Convert to screen coordinates
			turretSx, turretSy := r.camera.WorldToScreen(entity.X+mountX, entity.Y+mountY)

			// Draw turret as a circle and a line (barrel)
			turretRadius := 4.0 * r.camera.Zoom
			if turretRadius < 1.5 {
				turretRadius = 1.5
			}

			// Turret color (slightly lighter than ship)
			turretColor := color.RGBA{
				uint8(math.Min(255, float64(clr.R)+50)),
				uint8(math.Min(255, float64(clr.G)+50)),
				uint8(math.Min(255, float64(clr.B)+50)),
				255,
			}

			// Draw turret circle (base)
			vector.DrawFilledCircle(screen, float32(turretSx), float32(turretSy), float32(turretRadius), turretColor, true)

			// Draw turret outline circle for better visibility
			vector.StrokeCircle(screen, float32(turretSx), float32(turretSy), float32(turretRadius), 1.5, turretColor, true)

			// Draw turret barrel (line showing direction)
			// For player, use per-turret rotation; for others, use ship rotation + mount angle
			var turretRotation float64
			if entity.Type == EntityTypePlayer {
				if playerInput, ok := entity.Input.(*PlayerInput); ok {
					// Use per-turret rotation if available, fallback to ship rotation + mount angle
					turretRotation = playerInput.GetTurretRotation(turretIndex)
					if turretRotation == 0.0 {
						turretRotation = entity.Rotation + mount.Angle
					}
				} else {
					turretRotation = entity.Rotation + mount.Angle
				}
			} else {
				turretRotation = entity.Rotation + mount.Angle
			}

			// Barrel extends from center of turret circle
			// Use barrel length from mount point, or default to 3x turret radius if not set
			barrelLength := mount.BarrelLength * r.camera.Zoom
			if barrelLength == 0 {
				barrelLength = turretRadius * 3.0
			}
			barrelStartX := turretSx + math.Cos(turretRotation)*turretRadius
			barrelStartY := turretSy + math.Sin(turretRotation)*turretRadius
			barrelEndX := turretSx + math.Cos(turretRotation)*barrelLength
			barrelEndY := turretSy + math.Sin(turretRotation)*barrelLength

			// Draw barrel line (thicker for visibility)
			vector.StrokeLine(screen, float32(barrelStartX), float32(barrelStartY),
				float32(barrelEndX), float32(barrelEndY), 2.5, turretColor, true)
		}
	}

	// Draw aim target indicator for ships with turrets or shooting capability
	// Skip aim lines if there are too many entities to avoid performance issues
	if entity.Type != EntityTypeProjectile && drawAimLines {
		r.drawAimTarget(screen, entity, player)
	}

	// Draw health bar for damaged entities
	if entity.Health < entity.MaxHealth {
		barWidth := radius * 2
		barHeight := 4.0 * r.camera.Zoom
		barX := sx - barWidth/2
		barY := sy - radius - barHeight - 2

		// Background (red)
		vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(barWidth), float32(barHeight), color.RGBA{100, 0, 0, 255}, true)

		// Health (green)
		healthPercent := entity.Health / entity.MaxHealth
		healthWidth := barWidth * healthPercent
		vector.DrawFilledRect(screen, float32(barX), float32(barY), float32(healthWidth), float32(barHeight), color.RGBA{0, 255, 0, 255}, true)
	}
}

// drawAimTarget draws a line from the turret/shooting point to the target
func (r *Renderer) drawAimTarget(screen *ebiten.Image, entity *Entity, player *Entity) {
	var targetX, targetY float64
	var hasTarget bool
	var aimPointX, aimPointY float64

	shipConfig := GetShipTypeConfig(entity.ShipType)

	// Determine target based on entity type
	if entity.Type == EntityTypePlayer {
		// Player targets enemies - draw aim lines for each turret
		if playerInput, ok := entity.Input.(*PlayerInput); ok {
			// Draw aim line for each turret that has a target
			for turretIndex, mount := range shipConfig.TurretMounts {
				if !mount.Active {
					continue
				}
				turretTarget := playerInput.GetTurretTarget(turretIndex)
				if turretTarget.HasTarget {
					targetX = turretTarget.TargetX
					targetY = turretTarget.TargetY
					hasTarget = true

					// Calculate turret position for aim point
					cosRot := math.Cos(entity.Rotation)
					sinRot := math.Sin(entity.Rotation)
					mountX := mount.OffsetX*cosRot - mount.OffsetY*sinRot
					mountY := mount.OffsetX*sinRot + mount.OffsetY*cosRot
					aimPointX = entity.X + mountX
					aimPointY = entity.Y + mountY

					// Draw aim line for this turret with transparency
					aimSx, aimSy := r.camera.WorldToScreen(aimPointX, aimPointY)
					targetSx, targetSy := r.camera.WorldToScreen(targetX, targetY)
					r.drawTransparentLine(screen, aimSx, aimSy, targetSx, targetSy, color.RGBA{255, 255, 0, 30})

					targetRadius := 3.0 * r.camera.Zoom
					if targetRadius < 1.5 {
						targetRadius = 1.5
					}
					r.drawTransparentCircle(screen, targetSx, targetSy, targetRadius, color.RGBA{255, 255, 0, 30})
				}
			}
			// Return early since we've drawn all turret aim lines
			return
		}
	} else if entity.Type == EntityTypeEnemy {
		// Enemies target the player (with predictive aiming for shooters)
		// Skip drawing aim lines for homing enemies
		if player != nil && player.Active {
			// Check if this enemy has AI input with target info
			if aiInput, ok := entity.Input.(*AIInput); ok {
				// Only show aim lines for shooter enemies, not homing enemies
				if aiInput.EnemyType == EnemyTypeShooter {
					// Use stored target (which may be predictive for shooters)
					targetX = aiInput.TargetX
					targetY = aiInput.TargetY
					hasTarget = true

					// For enemies, aim from ship center (they shoot from center)
					aimPointX = entity.X
					aimPointY = entity.Y
				}
				// Skip homing enemies - don't set hasTarget
			} else {
				// Fallback: only show if we can't determine enemy type (shouldn't happen)
				// But skip to be safe
			}
		}
	}

	// Draw aim line if there's a target
	if hasTarget {
		// Convert to screen coordinates
		aimSx, aimSy := r.camera.WorldToScreen(aimPointX, aimPointY)
		targetSx, targetSy := r.camera.WorldToScreen(targetX, targetY)

		// Draw aim line with transparency
		aimColor := color.RGBA{255, 255, 0, 30} // Yellow, very transparent
		if entity.Type == EntityTypeEnemy {
			aimColor = color.RGBA{255, 100, 100, 30} // Light red for enemies, very transparent
		}

		// Draw line from aim point to target
		r.drawTransparentLine(screen, aimSx, aimSy, targetSx, targetSy, aimColor)

		// Draw small circle at target position
		targetRadius := 3.0 * r.camera.Zoom
		if targetRadius < 1.5 {
			targetRadius = 1.5
		}
		r.drawTransparentCircle(screen, targetSx, targetSy, targetRadius, aimColor)
	}
}

// drawTriangle draws an oblong triangle shape rotated by the entity's rotation
// The front point extends further to clearly show direction (arrowhead shape)
func (r *Renderer) drawTriangle(screen *ebiten.Image, x, y, radius, rotation float64, clr color.Color, shipType ShipType) {
	// Oblong triangle: front point extends further, back points form a wider base
	frontLength := radius * 1.5 // Front extends 1.5x the radius
	backOffset := radius * 0.5  // How far back the base is

	// Make homing enemies narrower
	backWidth := radius * 0.9 // Half-width of the back base (default)
	if shipType == ShipTypeHomingSuicide {
		backWidth = radius * 0.4 // Narrower for homing rockets
	}

	// Front point (extends forward)
	frontX := x + math.Cos(rotation)*frontLength
	frontY := y + math.Sin(rotation)*frontLength

	// Back left point (perpendicular to rotation direction, offset backward)
	backLeftX := x + math.Cos(rotation+math.Pi)*backOffset + math.Cos(rotation+math.Pi/2)*backWidth
	backLeftY := y + math.Sin(rotation+math.Pi)*backOffset + math.Sin(rotation+math.Pi/2)*backWidth

	// Back right point
	backRightX := x + math.Cos(rotation+math.Pi)*backOffset + math.Cos(rotation-math.Pi/2)*backWidth
	backRightY := y + math.Sin(rotation+math.Pi)*backOffset + math.Sin(rotation-math.Pi/2)*backWidth

	points := [3][2]float64{
		{frontX, frontY},         // Front point (tip)
		{backLeftX, backLeftY},   // Back left
		{backRightX, backRightY}, // Back right
	}

	// Draw triangle outline with thicker lines
	for i := 0; i < 3; i++ {
		next := (i + 1) % 3
		vector.StrokeLine(screen, float32(points[i][0]), float32(points[i][1]),
			float32(points[next][0]), float32(points[next][1]), 2, clr, true)
	}

	// Fill triangle by drawing lines from center to edges
	centerX, centerY := x, y
	for i := 0; i < 3; i++ {
		vector.StrokeLine(screen, float32(centerX), float32(centerY),
			float32(points[i][0]), float32(points[i][1]), 1, clr, true)
	}

	// Fill the back edge
	vector.StrokeLine(screen, float32(backLeftX), float32(backLeftY),
		float32(backRightX), float32(backRightY), 2, clr, true)
}

// drawSquare draws a square shape rotated by the entity's rotation
func (r *Renderer) drawSquare(screen *ebiten.Image, x, y, radius, rotation float64, clr color.Color) {
	// Square rotated by entity rotation
	halfSize := radius * 0.707 // radius * sqrt(2)/2 for diagonal
	points := [4][2]float64{
		{x + math.Cos(rotation+0.785)*halfSize, y + math.Sin(rotation+0.785)*halfSize}, // Top-right (45 degrees)
		{x + math.Cos(rotation+2.356)*halfSize, y + math.Sin(rotation+2.356)*halfSize}, // Bottom-right (135 degrees)
		{x + math.Cos(rotation+3.927)*halfSize, y + math.Sin(rotation+3.927)*halfSize}, // Bottom-left (225 degrees)
		{x + math.Cos(rotation+5.498)*halfSize, y + math.Sin(rotation+5.498)*halfSize}, // Top-left (315 degrees)
	}

	// Draw filled square by drawing triangles
	// Triangle 1: points 0, 1, 2
	vector.StrokeLine(screen, float32(points[0][0]), float32(points[0][1]),
		float32(points[1][0]), float32(points[1][1]), 2, clr, true)
	vector.StrokeLine(screen, float32(points[1][0]), float32(points[1][1]),
		float32(points[2][0]), float32(points[2][1]), 2, clr, true)
	vector.StrokeLine(screen, float32(points[2][0]), float32(points[2][1]),
		float32(points[0][0]), float32(points[0][1]), 2, clr, true)

	// Triangle 2: points 0, 2, 3
	vector.StrokeLine(screen, float32(points[0][0]), float32(points[0][1]),
		float32(points[2][0]), float32(points[2][1]), 2, clr, true)
	vector.StrokeLine(screen, float32(points[2][0]), float32(points[2][1]),
		float32(points[3][0]), float32(points[3][1]), 2, clr, true)
	vector.StrokeLine(screen, float32(points[3][0]), float32(points[3][1]),
		float32(points[0][0]), float32(points[0][1]), 2, clr, true)

	// Fill by drawing lines from center
	centerX, centerY := x, y
	for i := 0; i < 4; i++ {
		vector.StrokeLine(screen, float32(centerX), float32(centerY),
			float32(points[i][0]), float32(points[i][1]), 1, clr, true)
	}
}

// drawDiamond draws a diamond shape rotated by the entity's rotation
func (r *Renderer) drawDiamond(screen *ebiten.Image, x, y, radius, rotation float64, clr color.Color) {
	// Diamond (square rotated 45 degrees) pointing forward
	points := [4][2]float64{
		{x + math.Cos(rotation)*radius, y + math.Sin(rotation)*radius},             // Front point
		{x + math.Cos(rotation+1.571)*radius, y + math.Sin(rotation+1.571)*radius}, // Right point (90 degrees)
		{x + math.Cos(rotation+3.142)*radius, y + math.Sin(rotation+3.142)*radius}, // Back point (180 degrees)
		{x + math.Cos(rotation+4.712)*radius, y + math.Sin(rotation+4.712)*radius}, // Left point (270 degrees)
	}

	// Draw diamond outline
	for i := 0; i < 4; i++ {
		next := (i + 1) % 4
		vector.StrokeLine(screen, float32(points[i][0]), float32(points[i][1]),
			float32(points[next][0]), float32(points[next][1]), 2, clr, true)
	}

	// Fill diamond (draw lines from center to each point)
	centerX, centerY := x, y
	for i := 0; i < 4; i++ {
		vector.StrokeLine(screen, float32(centerX), float32(centerY),
			float32(points[i][0]), float32(points[i][1]), 1, clr, true)
	}
}

// RenderUI renders the user interface (score, FPS, restart message, etc.)
func (r *Renderer) RenderUI(screen *ebiten.Image, player *Entity, score int, fps float64) {
	// Always show score
	scoreText := fmt.Sprintf("Score: %d", score)
	r.drawText(screen, scoreText, 10, 30, color.RGBA{255, 255, 255, 255})

	// Always show FPS
	fpsText := fmt.Sprintf("FPS: %.0f", fps)
	r.drawText(screen, fpsText, 10, 50, color.RGBA{200, 200, 200, 255})

	// Show restart message if player is dead
	if player == nil || !player.Active || player.Health <= 0 {
		restartText := "[R] to Restart"
		textWidth := r.measureText(restartText)
		textX := (r.camera.Width - textWidth) / 2
		textY := r.camera.Height / 2
		r.drawText(screen, restartText, textX, textY, color.RGBA{255, 255, 0, 255})
	}
}

// drawText draws text on the screen
func (r *Renderer) drawText(screen *ebiten.Image, str string, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	face := &text.GoTextFace{
		Source: r.faceSource,
		Size:   16,
	}
	text.Draw(screen, str, face, op)
}

// measureText measures the width of text
func (r *Renderer) measureText(str string) float64 {
	face := &text.GoTextFace{
		Source: r.faceSource,
		Size:   16,
	}
	_, advance := text.Measure(str, face, 0)
	return advance
}

// drawTransparentLine draws a line with proper alpha transparency
func (r *Renderer) drawTransparentLine(screen *ebiten.Image, x1, y1, x2, y2 float64, clr color.RGBA) {
	// Calculate line bounds
	minX := math.Min(x1, x2) - 2
	maxX := math.Max(x1, x2) + 2
	minY := math.Min(y1, y2) - 2
	maxY := math.Max(y1, y2) + 2

	width := int(maxX - minX)
	height := int(maxY - minY)
	if width <= 0 || height <= 0 {
		return
	}

	// Create temporary image for the line
	lineImg := ebiten.NewImage(width, height)

	// Draw line on temporary image
	lineX1 := float32(x1 - minX)
	lineY1 := float32(y1 - minY)
	lineX2 := float32(x2 - minX)
	lineY2 := float32(y2 - minY)
	vector.StrokeLine(lineImg, lineX1, lineY1, lineX2, lineY2, 1.5, clr, true)

	// Draw temporary image to screen with alpha
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(minX, minY)
	op.ColorM.Scale(1, 1, 1, float64(clr.A)/255.0)
	screen.DrawImage(lineImg, op)
}

// drawTransparentCircle draws a circle outline with proper alpha transparency
func (r *Renderer) drawTransparentCircle(screen *ebiten.Image, x, y, radius float64, clr color.RGBA) {
	// Create temporary image for the circle
	size := int(radius*2 + 4)
	if size <= 0 {
		return
	}
	circleImg := ebiten.NewImage(size, size)

	// Draw circle on temporary image
	centerX := float32(radius + 2)
	centerY := float32(radius + 2)
	vector.StrokeCircle(circleImg, centerX, centerY, float32(radius), 1.5, clr, true)

	// Draw temporary image to screen with alpha
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x-radius-2, y-radius-2)
	op.ColorM.Scale(1, 1, 1, float64(clr.A)/255.0)
	screen.DrawImage(circleImg, op)
}
