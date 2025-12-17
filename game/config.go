package game

// Config holds game configuration constants
type Config struct {
	// CellSize is the size of each spatial partition cell in pixels
	CellSize float64

	// WorldMinX is the minimum X coordinate of the world
	WorldMinX float64

	// WorldMinY is the minimum Y coordinate of the world
	WorldMinY float64

	// WorldWidth is the total width of the game world in pixels
	WorldWidth float64

	// WorldHeight is the total height of the game world in pixels
	WorldHeight float64

	// ScreenWidth is the window width in pixels
	ScreenWidth int

	// ScreenHeight is the window height in pixels
	ScreenHeight int
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		CellSize:     2048.0,
		WorldMinX:    -100000.0,
		WorldMinY:    -100000.0,
		WorldWidth:   200000.0, // From -100000 to 100000
		WorldHeight:  200000.0, // From -100000 to 100000
		ScreenWidth:  1024,
		ScreenHeight: 768,
	}
}

// CellCountX returns the number of cells in the X direction
func (c Config) CellCountX() int {
	return int(c.WorldWidth / c.CellSize)
}

// CellCountY returns the number of cells in the Y direction
func (c Config) CellCountY() int {
	return int(c.WorldHeight / c.CellSize)
}
