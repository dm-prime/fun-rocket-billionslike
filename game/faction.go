package game

// Faction represents which side an entity belongs to
type Faction int

const (
	FactionPlayer Faction = iota
	FactionEnemy
)

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
