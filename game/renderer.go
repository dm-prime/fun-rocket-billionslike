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

	// Determine color based on entity type
	var clr color.Color
	switch entity.Type {
	case EntityTypePlayer:
		clr = color.RGBA{0, 255, 0, 255} // Green
	case EntityTypeEnemy:
		// Different colors for different enemy types
		if aiInput, ok := entity.Input.(*AIInput); ok {
			switch aiInput.EnemyType {
			case EnemyTypeHomingSuicide:
				clr = color.RGBA{255, 100, 0, 255} // Orange (suicide)
			case EnemyTypeShooter:
				clr = color.RGBA{255, 0, 0, 255} // Red (shooter)
			default:
				clr = color.RGBA{255, 0, 0, 255} // Red (default)
			}
		} else {
			clr = color.RGBA{255, 0, 0, 255} // Red (default)
		}
	case EntityTypeProjectile:
		clr = color.RGBA{255, 255, 0, 255} // Yellow
	default:
		clr = color.RGBA{255, 255, 255, 255} // White
	}

	// Draw entity as a circle
	radius := entity.Radius * r.camera.Zoom
	if radius < 1 {
		radius = 1
	}

	// Draw filled circle
	vector.DrawFilledCircle(screen, float32(sx), float32(sy), float32(radius), clr, true)

	// Draw direction indicator (small line)
	if entity.Type != EntityTypeProjectile {
		dirLength := radius * 1.5
		endX := sx + math.Cos(entity.Rotation)*dirLength
		endY := sy + math.Sin(entity.Rotation)*dirLength
		vector.StrokeLine(screen, float32(sx), float32(sy), float32(endX), float32(endY), 2, clr, true)
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

