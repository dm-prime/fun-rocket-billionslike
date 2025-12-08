package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	camera *Camera
}

// NewRenderer creates a new renderer
func NewRenderer(camera *Camera) *Renderer {
	return &Renderer{
		camera: camera,
	}
}

// Render renders all visible entities
func (r *Renderer) Render(screen *ebiten.Image, world *World) {
	// Get visible cells
	visibleCells := r.camera.GetVisibleCells(world)

	// Render entities in visible cells
	for _, cell := range visibleCells {
		for _, entity := range cell.GetActiveEntities() {
			if entity.Health <= 0 {
				continue
			}
			r.RenderEntity(screen, entity)
		}
	}
}

// RenderEntity renders a single entity
func (r *Renderer) RenderEntity(screen *ebiten.Image, entity *Entity) {
	// Convert world coordinates to screen coordinates
	sx, sy := r.camera.WorldToScreen(entity.X, entity.Y)

	// Skip if outside screen bounds (with margin)
	margin := 100.0
	if sx < -margin || sx > r.camera.Width+margin ||
		sy < -margin || sy > r.camera.Height+margin {
		return
	}

	// Determine color based on ship type
	var clr color.Color
	if entity.Type == EntityTypeProjectile {
		clr = color.RGBA{255, 255, 0, 255} // Yellow
	} else {
		// Use ship type for color
		shipConfig := GetShipTypeConfig(entity.ShipType)
		clr = shipConfig.Color
	}

	// Draw entity based on ship shape
	radius := entity.Radius * r.camera.Zoom
	if radius < 1 {
		radius = 1
	}

	// Get ship config for shape
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
		r.drawTriangle(screen, sx, sy, radius, entity.Rotation, clr)
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
	if entity.Type != EntityTypeProjectile {
		shipConfig := GetShipTypeConfig(entity.ShipType)
		for _, mount := range shipConfig.TurretMounts {
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
			var turretColor color.RGBA
			if rgba, ok := clr.(color.RGBA); ok {
				turretColor = color.RGBA{
					uint8(math.Min(255, float64(rgba.R)+50)),
					uint8(math.Min(255, float64(rgba.G)+50)),
					uint8(math.Min(255, float64(rgba.B)+50)),
					255,
				}
			} else {
				turretColor = color.RGBA{200, 200, 200, 255} // Default gray
			}
			
			// Draw turret circle (base)
			vector.DrawFilledCircle(screen, float32(turretSx), float32(turretSy), float32(turretRadius), turretColor, true)
			
			// Draw turret outline circle for better visibility
			vector.StrokeCircle(screen, float32(turretSx), float32(turretSy), float32(turretRadius), 1.5, turretColor, true)
			
			// Draw turret barrel (line showing direction)
			// For player, use turret rotation; for others, use ship rotation + mount angle
			var turretRotation float64
			if entity.Type == EntityTypePlayer {
				if playerInput, ok := entity.Input.(*PlayerInput); ok {
					turretRotation = playerInput.TurretRotation
				} else {
					turretRotation = entity.Rotation + mount.Angle
				}
			} else {
				turretRotation = entity.Rotation + mount.Angle
			}
			
			// Barrel extends from center of turret circle
			barrelLength := turretRadius * 3.0
			barrelStartX := turretSx + math.Cos(turretRotation)*turretRadius
			barrelStartY := turretSy + math.Sin(turretRotation)*turretRadius
			barrelEndX := turretSx + math.Cos(turretRotation)*barrelLength
			barrelEndY := turretSy + math.Sin(turretRotation)*barrelLength
			
			// Draw barrel line (thicker for visibility)
			vector.StrokeLine(screen, float32(barrelStartX), float32(barrelStartY),
				float32(barrelEndX), float32(barrelEndY), 2.5, turretColor, true)
		}
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

// drawTriangle draws an oblong triangle shape rotated by the entity's rotation
// The front point extends further to clearly show direction (arrowhead shape)
func (r *Renderer) drawTriangle(screen *ebiten.Image, x, y, radius, rotation float64, clr color.Color) {
	// Oblong triangle: front point extends further, back points form a wider base
	frontLength := radius * 1.5  // Front extends 1.5x the radius
	backOffset := radius * 0.5    // How far back the base is
	backWidth := radius * 0.9     // Half-width of the back base
	
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
		{frontX, frontY},           // Front point (tip)
		{backLeftX, backLeftY},     // Back left
		{backRightX, backRightY},   // Back right
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
		{x + math.Cos(rotation)*radius, y + math.Sin(rotation)*radius},           // Front point
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

