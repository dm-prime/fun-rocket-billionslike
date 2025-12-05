package main

import "math"

// updateDust updates the dust particle system relative to player movement
func (g *Game) updateDust(dt float64, player *Ship) {
	// Move dust relative to ship velocity (opposite direction for parallax effect)
	span := math.Hypot(float64(screenWidth), float64(screenHeight)) * dustSpanMultiplier
	half := span * 0.5
	for i := range g.dust {
		// Dust moves opposite to ship movement, with individual speed variance
		speedMultiplier := g.dust[i].speed
		g.dust[i].pos.x -= player.vel.x * dt * speedMultiplier
		g.dust[i].pos.y -= player.vel.y * dt * speedMultiplier

		// Keep dust in a torus around the player so they don't depend on absolute origin.
		dx := g.dust[i].pos.x - player.pos.x
		dy := g.dust[i].pos.y - player.pos.y

		if dx < -half {
			g.dust[i].pos.x += span
		}
		if dx > half {
			g.dust[i].pos.x -= span
		}
		if dy < -half {
			g.dust[i].pos.y += span
		}
		if dy > half {
			g.dust[i].pos.y -= span
		}
	}
}


