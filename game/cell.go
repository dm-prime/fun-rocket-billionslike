package game

// Cell represents a spatial partition cell containing entities
type Cell struct {
	// Entities in this cell (preallocated slice)
	Entities []*Entity

	// Current count of active entities
	Count int
}

// NewCell creates a new cell with preallocated entity storage
func NewCell(initialCapacity int) *Cell {
	return &Cell{
		Entities: make([]*Entity, 0, initialCapacity),
		Count:    0,
	}
}

// AddEntity adds an entity to this cell
func (c *Cell) AddEntity(entity *Entity) {
	// Check if entity already in cell
	for i := 0; i < c.Count; i++ {
		if c.Entities[i] == entity {
			return // Already in cell
		}
	}

	// Add entity
	if c.Count < len(c.Entities) {
		c.Entities[c.Count] = entity
	} else {
		c.Entities = append(c.Entities, entity)
	}
	c.Count++
}

// RemoveEntity removes an entity from this cell
func (c *Cell) RemoveEntity(entity *Entity) {
	for i := 0; i < c.Count; i++ {
		if c.Entities[i] == entity {
			// Swap with last element and decrease count
			c.Entities[i] = c.Entities[c.Count-1]
			c.Entities[c.Count-1] = nil
			c.Count--
			return
		}
	}
}

// GetEntities returns all active entities in this cell
func (c *Cell) GetEntities() []*Entity {
	return c.Entities[:c.Count]
}

// Clear removes all entities from the cell (but keeps capacity)
func (c *Cell) Clear() {
	for i := 0; i < c.Count; i++ {
		c.Entities[i] = nil
	}
	c.Count = 0
}

// GetActiveEntities returns only active entities
func (c *Cell) GetActiveEntities() []*Entity {
	active := make([]*Entity, 0, c.Count)
	for i := 0; i < c.Count; i++ {
		if c.Entities[i].Active {
			active = append(active, c.Entities[i])
		}
	}
	return active
}

