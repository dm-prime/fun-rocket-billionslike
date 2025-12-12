package game

// World manages the spatial partitioning grid and entity registration
type World struct {
	// Preallocated 2D grid of cells
	Cells [][]*Cell

	// Configuration
	Config Config

	// All entities in the world (for iteration)
	AllEntities []*Entity

	// Entity pool for reuse
	EntityPool []*Entity
	PoolIndex  int
}

// NewWorld creates a new world with preallocated cells
func NewWorld(config Config) *World {
	cellCountX := config.CellCountX()
	cellCountY := config.CellCountY()

	// Preallocate 2D grid
	cells := make([][]*Cell, cellCountX)
	for x := 0; x < cellCountX; x++ {
		cells[x] = make([]*Cell, cellCountY)
		for y := 0; y < cellCountY; y++ {
			// Preallocate each cell with initial capacity
			cells[x][y] = NewCell(100)
		}
	}

	return &World{
		Cells:       cells,
		Config:      config,
		AllEntities: make([]*Entity, 0, 10000),
		EntityPool:  make([]*Entity, 0, 1000),
		PoolIndex:   0,
	}
}

// WorldToCell converts world coordinates to cell coordinates
func (w *World) WorldToCell(x, y float64) (int, int) {
	cellX := int(x / w.Config.CellSize)
	cellY := int(y / w.Config.CellSize)

	// Clamp to valid cell range
	cellX = max(0, min(cellX, w.Config.CellCountX()-1))
	cellY = max(0, min(cellY, w.Config.CellCountY()-1))

	return cellX, cellY
}

// GetCell returns the cell at the given cell coordinates
func (w *World) GetCell(cellX, cellY int) *Cell {
	if cellX < 0 || cellX >= w.Config.CellCountX() ||
		cellY < 0 || cellY >= w.Config.CellCountY() {
		return nil
	}
	return w.Cells[cellX][cellY]
}

// RegisterEntity adds an entity to the world and assigns it to the correct cell
func (w *World) RegisterEntity(entity *Entity) {
	// Calculate cell coordinates
	cellX, cellY := w.WorldToCell(entity.X, entity.Y)
	entity.CellX = cellX
	entity.CellY = cellY

	// Add to cell
	cell := w.GetCell(cellX, cellY)
	if cell != nil {
		cell.AddEntity(entity)
	}

	// Add to all entities list
	w.AllEntities = append(w.AllEntities, entity)
}

// UnregisterEntity removes an entity from the world
func (w *World) UnregisterEntity(entity *Entity) {
	// Remove from cell
	cell := w.GetCell(entity.CellX, entity.CellY)
	if cell != nil {
		cell.RemoveEntity(entity)
	}

	// Remove from all entities list
	for i, e := range w.AllEntities {
		if e == entity {
			w.AllEntities[i] = w.AllEntities[len(w.AllEntities)-1]
			w.AllEntities = w.AllEntities[:len(w.AllEntities)-1]
			break
		}
	}
}

// UpdateEntityCell updates an entity's cell membership if it moved
func (w *World) UpdateEntityCell(entity *Entity) {
	newCellX, newCellY := w.WorldToCell(entity.X, entity.Y)

	// If entity moved to a different cell, update cell membership
	if newCellX != entity.CellX || newCellY != entity.CellY {
		// Remove from old cell
		oldCell := w.GetCell(entity.CellX, entity.CellY)
		if oldCell != nil {
			oldCell.RemoveEntity(entity)
		}

		// Add to new cell
		entity.CellX = newCellX
		entity.CellY = newCellY
		newCell := w.GetCell(newCellX, newCellY)
		if newCell != nil {
			newCell.AddEntity(entity)
		}
	}
}

// GetCellsForEntity returns all adjacent cells (3x3 grid) for collision checking
// We always return the full 3x3 grid because entities are only stored in their center cell,
// but we need to check adjacent cells to catch collisions with entities near cell boundaries.
func (w *World) GetCellsForEntity(entity *Entity) []*Cell {
	cells := make([]*Cell, 0, 9) // Max 9 cells (3x3 grid)

	// Get center cell
	centerX, centerY := w.WorldToCell(entity.X, entity.Y)

	// Always check center cell and all adjacent cells (3x3 grid)
	// This is necessary because entities are only stored in their center cell,
	// so a small entity near a cell boundary needs to check adjacent cells
	// to find entities whose center is in those cells but could still collide.
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			cellX := centerX + dx
			cellY := centerY + dy

			cell := w.GetCell(cellX, cellY)
			if cell != nil {
				cells = append(cells, cell)
			}
		}
	}

	return cells
}

// GetEntitiesInRadius returns all entities within a radius of a point
func (w *World) GetEntitiesInRadius(x, y, radius float64) []*Entity {
	entities := make([]*Entity, 0, 100)

	// Get cells that might contain entities in radius
	minCellX, minCellY := w.WorldToCell(x-radius, y-radius)
	maxCellX, maxCellY := w.WorldToCell(x+radius, y+radius)

	for cellX := minCellX; cellX <= maxCellX; cellX++ {
		for cellY := minCellY; cellY <= maxCellY; cellY++ {
			cell := w.GetCell(cellX, cellY)
			if cell == nil {
				continue
			}

			// Optimize: iterate directly over cell entities to avoid GetActiveEntities allocation
			for i := 0; i < cell.Count; i++ {
				entity := cell.Entities[i]
				if !entity.Active {
					continue
				}
				dx := entity.X - x
				dy := entity.Y - y
				distanceSq := dx*dx + dy*dy
				radiusSq := radius * radius
				if distanceSq <= radiusSq {
					entities = append(entities, entity)
				}
			}
		}
	}

	return entities
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
