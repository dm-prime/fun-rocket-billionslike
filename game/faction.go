package game

import "image/color"

// Faction represents which side an entity belongs to
type Faction int

const (
	FactionPlayer Faction = iota
	FactionEnemy
)

// FactionConfig holds configuration for each faction
type FactionConfig struct {
	Faction Faction
	Color   color.RGBA
}

var (
	// FactionConfigs holds configuration for each faction
	FactionConfigs = map[Faction]FactionConfig{
		FactionPlayer: {
			Faction: FactionPlayer,
			Color:   color.RGBA{0, 255, 0, 255}, // Green for player faction
		},
		FactionEnemy: {
			Faction: FactionEnemy,
			Color:   color.RGBA{255, 0, 0, 255}, // Red for enemy faction
		},
	}
)

// GetFactionConfig returns configuration for a faction
func GetFactionConfig(faction Faction) FactionConfig {
	if config, ok := FactionConfigs[faction]; ok {
		return config
	}
	// Default fallback
	return FactionConfig{
		Faction: faction,
		Color:   color.RGBA{255, 100, 0, 255}, // Orange fallback
	}
}

// GetOppositeFaction returns the opposite faction for targeting purposes
func GetOppositeFaction(faction Faction) Faction {
	switch faction {
	case FactionPlayer:
		return FactionEnemy
	case FactionEnemy:
		return FactionPlayer
	default:
		return FactionEnemy
	}
}

// GetEntityFaction returns the faction of an entity
func GetEntityFaction(entity *Entity) Faction {
	if entity == nil {
		return FactionEnemy
	}
	return entity.Faction
}
