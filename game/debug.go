package game

// DebugState holds global debug flags that persist across game resets
type DebugState struct {
	ShowGrid bool // Show cell grid lines and cell coordinates
}

// Global debug state instance (persists across game resets)
var globalDebugState = &DebugState{
	ShowGrid: false, // Default to off
}

// GetDebugState returns the global debug state
func GetDebugState() *DebugState {
	return globalDebugState
}



