package main

import "image/color"

// initFactions seeds faction colors and alliance relationships.
func (g *Game) initFactions() {
	g.factionColors = map[string]color.NRGBA{
		"Union":   {R: 180, G: 220, B: 255, A: 255},
		"Raiders": {R: 220, G: 40, B: 40, A: 255},
		"Traders": {R: 80, G: 200, B: 120, A: 255},
	}

	g.alliances = make(map[string]map[string]bool)
	g.setAlliance("Union", "Union")
	g.setAlliance("Raiders", "Raiders")
	g.setAlliance("Traders", "Traders")
	g.setAlliance("Union", "Traders") // Player-aligned faction for future friendly ships.
}

func (g *Game) setAlliance(a, b string) {
	if g.alliances[a] == nil {
		g.alliances[a] = map[string]bool{}
	}
	if g.alliances[b] == nil {
		g.alliances[b] = map[string]bool{}
	}
	g.alliances[a][b] = true
	g.alliances[b][a] = true
}

func (g *Game) areAllied(factionA, factionB string) bool {
	if factionA == "" || factionB == "" {
		return false
	}
	if factionA == factionB {
		return true
	}
	if allies, ok := g.alliances[factionA]; ok {
		if allies[factionB] {
			return true
		}
	}
	if allies, ok := g.alliances[factionB]; ok {
		if allies[factionA] {
			return true
		}
	}
	return false
}

func (g *Game) colorForFaction(faction string) color.NRGBA {
	if c, ok := g.factionColors[faction]; ok {
		return c
	}
	return color.NRGBA{R: 200, G: 200, B: 200, A: 255}
}

