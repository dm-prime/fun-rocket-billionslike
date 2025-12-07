package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// drawRadarTrail draws a trail segment on the radar with proper transformations and fading
func (g *Game) drawRadarTrail(screen *ebiten.Image, trail []RadarTrailPoint, trailColor color.NRGBA, player *Ship, center vec2, scale float64, radarRadius float64) {
	if len(trail) <= 1 {
		return
	}

	edgeLimit := radarRadius - radarEdgeMargin
	for j := 0; j < len(trail)-1; j++ {
		p1 := trail[j]
		p2 := trail[j+1]

		// Transform world coordinates to radar coordinates (relative to current player position)
		dx1 := p1.pos.x - player.pos.x
		dy1 := p1.pos.y - player.pos.y
		rotated1 := rotatePoint(vec2{dx1, dy1}, -player.angle)
		rx1 := rotated1.x * scale
		ry1 := rotated1.y * scale

		dx2 := p2.pos.x - player.pos.x
		dy2 := p2.pos.y - player.pos.y
		rotated2 := rotatePoint(vec2{dx2, dy2}, -player.angle)
		rx2 := rotated2.x * scale
		ry2 := rotated2.y * scale

		// Clamp to radar edge if needed
		if edgeDist1 := math.Hypot(rx1, ry1); edgeDist1 > edgeLimit {
			f := edgeLimit / edgeDist1
			rx1 *= f
			ry1 *= f
		}
		if edgeDist2 := math.Hypot(rx2, ry2); edgeDist2 > edgeLimit {
			f := edgeLimit / edgeDist2
			rx2 *= f
			ry2 *= f
		}

		// Calculate opacity based on age (fade from full to transparent)
		age := (p1.age + p2.age) / 2.0
		opacity := clamp(1.0-(age/radarTrailMaxAge), 0.0, 1.0)

		// Create faded color
		fadedColor := color.NRGBA{
			R: trailColor.R,
			G: trailColor.G,
			B: trailColor.B,
			A: uint8(float64(trailColor.A) * opacity * trailOpacityMax),
		}

		// Draw trail segment
		ebitenutil.DrawLine(
			screen,
			center.x+rx1,
			center.y+ry1,
			center.x+rx2,
			center.y+ry2,
			fadedColor,
		)
	}
}

// updateRadarTrails updates the trail points for all ships on the radar
func (g *Game) updateRadarTrails(dt float64, player *Ship) {
	for id, ship := range g.ships {
		// Initialize timer if needed
		if _, exists := g.radarTrailTimers[id]; !exists {
			g.radarTrailTimers[id] = 0
		}

		// Age existing trail points
		trail := g.radarTrails[id]
		for j := range trail {
			trail[j].age += dt
		}

		// Remove old trail points
		newTrail := make([]RadarTrailPoint, 0, len(trail))
		for _, point := range trail {
			if point.age < radarTrailMaxAge {
				newTrail = append(newTrail, point)
			}
		}
		g.radarTrails[id] = newTrail

		// Add new trail point periodically
		g.radarTrailTimers[id] += dt
		if g.radarTrailTimers[id] >= radarTrailUpdateInterval {
			// Add new point with world coordinates
			newPoint := RadarTrailPoint{pos: ship.pos, age: 0}
			g.radarTrails[id] = append(g.radarTrails[id], newPoint)

			// Limit trail length
			if len(g.radarTrails[id]) > radarTrailMaxPoints {
				g.radarTrails[id] = g.radarTrails[id][1:]
			}

			g.radarTrailTimers[id] = 0
		}
	}
}

// drawRadar renders a simple orientable radar centered on the player ship showing nearby enemies.
func (g *Game) drawRadar(screen *ebiten.Image, player *Ship) {
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}
	center := screenCenter

	// Radar backdrop
	radarRadius := getRadarRadius()
	drawCircle(screen, center.x, center.y, radarRadius+radarEdgeMargin, colorRadarBackdrop)
	drawCircle(screen, center.x, center.y, radarRadius, colorRadarRing)

	// Player heading marker (always points up since radar rotates with player view)
	headingLen := radarRadius - radarHeadingOffset
	headX := center.x
	headY := center.y - headingLen
	ebitenutil.DrawLine(screen, center.x, center.y, headX, headY, colorRadarHeading)
	drawCircle(screen, center.x, center.y, radarCenterDotSize, colorRadarPlayer)

	// Rotated radar (matches game rotation style). Rotate enemy positions relative to player angle.
	scale := radarRadius / radarRange

	// Draw player trail
	g.drawRadarTrail(screen, g.radarTrails[g.playerID], colorRadarPlayer, player, center, scale, radarRadius)

	// Draw player predictive trail
	playerInput := getPlayerInput()
	playerPredictedPositions := g.predictFuturePath(player, playerInput)
	g.drawPredictiveTrailInRadar(screen, playerPredictedPositions, player, player, center, scale, radarRadius, colorRadarPlayer)

	// Collect all radar blip data
	type radarBlip struct {
		entityID       EntityID
		isRock         bool
		rx, ry         float64 // radar coordinates
		dist           float64
		blipColor      color.NRGBA
		isOffRadar     bool
		dirX, dirY     float64 // direction for off-radar blips
		labelX, labelY float64 // label position
		label          string
	}

	blips := make([]radarBlip, 0)
	
	// Add ship blips
	for id, enemy := range g.ships {
		if id == g.playerID {
			continue
		}

		dx := enemy.pos.x - player.pos.x
		dy := enemy.pos.y - player.pos.y

		dist := math.Hypot(dx, dy)

		// Rotate the offset relative to player angle (same as ship rendering)
		rotated := rotatePoint(vec2{dx, dy}, -player.angle)
		rx := rotated.x * scale
		ry := rotated.y * scale

		blipColor := g.colorForFaction(enemy.faction)
		isOffRadar := dist > radarRange
		var labelX, labelY, dirX, dirY float64
		var label string

		if isOffRadar {
			// Place on the edge of the radar circle and show distance
			dirX = rotated.x / dist
			dirY = rotated.y / dist
			maxR := radarRadius - 5
			rx = dirX * maxR
			ry = dirY * maxR

			label = fmt.Sprintf("%.0f", dist)
			labelX = center.x + rx + dirX*radarOffRadarDist
			labelY = center.y + ry + dirY*radarOffRadarDist
			minX := center.x - radarRadius + 6
			maxX := center.x + radarRadius - 32
			minY := center.y - radarRadius + 6
			maxY := center.y + radarRadius - 12
			if labelX < minX {
				labelX = minX
			}
			if labelX > maxX {
				labelX = maxX
			}
			if labelY < minY {
				labelY = minY
			}
			if labelY > maxY {
				labelY = maxY
			}
		} else {
			// Clamp to radar edge so distant targets sit on the rim
			edgeLimit := radarRadius - radarEdgeMargin
			if edgeDist := math.Hypot(rx, ry); edgeDist > edgeLimit {
				f := edgeLimit / edgeDist
				rx *= f
				ry *= f
			}
			// For on-radar blips, show distance label near the dot
			label = fmt.Sprintf("%.0f", dist)
			labelX = center.x + rx + radarLabelOffsetX
			labelY = center.y + ry - radarLabelOffsetY
		}

		blips = append(blips, radarBlip{
			entityID:   id,
			isRock:     false,
			rx:         rx,
			ry:         ry,
			dist:       dist,
			blipColor:  blipColor,
			isOffRadar: isOffRadar,
			dirX:       dirX,
			dirY:       dirY,
			labelX:     labelX,
			labelY:     labelY,
			label:      label,
		})
	}
	
	// Add rock blips (only if on collision course)
	for id, rock := range g.rocks {
		// Only show on radar if on collision course
		if !g.isOnCollisionCourse(player, rock, collisionCourseLookAhead) {
			continue
		}

		dx := rock.pos.x - player.pos.x
		dy := rock.pos.y - player.pos.y

		dist := math.Hypot(dx, dy)

		// Rotate the offset relative to player angle (same as ship rendering)
		rotated := rotatePoint(vec2{dx, dy}, -player.angle)
		rx := rotated.x * scale
		ry := rotated.y * scale

		blipColor := g.colorForFaction("Rocks")
		isOffRadar := dist > radarRange
		var labelX, labelY, dirX, dirY float64
		var label string

		if isOffRadar {
			// Place on the edge of the radar circle and show distance
			dirX = rotated.x / dist
			dirY = rotated.y / dist
			maxR := radarRadius - 5
			rx = dirX * maxR
			ry = dirY * maxR

			label = fmt.Sprintf("%.0f", dist)
			labelX = center.x + rx + dirX*radarOffRadarDist
			labelY = center.y + ry + dirY*radarOffRadarDist
			minX := center.x - radarRadius + 6
			maxX := center.x + radarRadius - 32
			minY := center.y - radarRadius + 6
			maxY := center.y + radarRadius - 12
			if labelX < minX {
				labelX = minX
			}
			if labelX > maxX {
				labelX = maxX
			}
			if labelY < minY {
				labelY = minY
			}
			if labelY > maxY {
				labelY = maxY
			}
		} else {
			// Clamp to radar edge so distant targets sit on the rim
			edgeLimit := radarRadius - radarEdgeMargin
			if edgeDist := math.Hypot(rx, ry); edgeDist > edgeLimit {
				f := edgeLimit / edgeDist
				rx *= f
				ry *= f
			}
			// For on-radar blips, show distance label near the dot
			label = fmt.Sprintf("%.0f", dist)
			labelX = center.x + rx + radarLabelOffsetX
			labelY = center.y + ry - radarLabelOffsetY
		}

		blips = append(blips, radarBlip{
			entityID:   id,
			isRock:     true,
			rx:         rx,
			ry:         ry,
			dist:       dist,
			blipColor:  blipColor,
			isOffRadar: isOffRadar,
			dirX:       dirX,
			dirY:       dirY,
			labelX:     labelX,
			labelY:     labelY,
			label:      label,
		})
	}

	// Group blips that are close together
	type cluster struct {
		blips            []*radarBlip
		centerX, centerY float64
	}
	clusters := make([]cluster, 0)
	assigned := make(map[int]bool)

	for i := range blips {
		if assigned[i] {
			continue
		}
		// Start a new cluster with this blip
		clust := cluster{
			blips:   []*radarBlip{&blips[i]},
			centerX: blips[i].rx,
			centerY: blips[i].ry,
		}
		assigned[i] = true

		// Find all blips close to this one
		for j := range blips {
			if assigned[j] || i == j {
				continue
			}
			dist := math.Hypot(blips[i].rx-blips[j].rx, blips[i].ry-blips[j].ry)
			if dist < radarStackThreshold {
				clust.blips = append(clust.blips, &blips[j])
				assigned[j] = true
			}
		}

		// Calculate cluster center
		if len(clust.blips) > 1 {
			sumX, sumY := 0.0, 0.0
			for _, b := range clust.blips {
				sumX += b.rx
				sumY += b.ry
			}
			clust.centerX = sumX / float64(len(clust.blips))
			clust.centerY = sumY / float64(len(clust.blips))
		}

		clusters = append(clusters, clust)
	}

	// Draw trails first (before dots) - only for ships, not rocks
	for id, ship := range g.ships {
		if id == g.playerID {
			continue
		}
		trailColor := g.colorForFaction(ship.faction)
		g.drawRadarTrail(screen, g.radarTrails[id], trailColor, player, center, scale, radarRadius)
	}

	// Draw dots, labels, and indicators with stacking
	for _, clust := range clusters {
		if len(clust.blips) == 1 {
			// Single blip, draw normally
			b := clust.blips[0]
			baseX := center.x + b.rx
			baseY := center.y + b.ry

			// Draw indicators first (so they appear behind the dot) - only for ships, not rocks
			if !b.isRock {
				if ship := g.GetShip(b.entityID); ship != nil {
					g.drawRadarIndicators(screen, ship, b.entityID, baseX, baseY, b.blipColor, player, scale, radarRadius, center)
				}
			}

			// Draw dot
			drawCircle(screen, baseX, baseY, radarBlipSize, b.blipColor)
			ebitenutil.DebugPrintAt(screen, b.label, int(b.labelX), int(b.labelY))

			// Draw state label below distance - only for ships, not rocks
			if !b.isRock {
				state := g.getNPCState(b.entityID)
				stateLabel := g.getNPCStateString(state)
				ebitenutil.DebugPrintAt(screen, stateLabel, int(b.labelX), int(b.labelY)+12)
			}
		} else {
			// Multiple blips, stack them vertically
			// Sort by distance (closest first, so it's at the bottom of the stack)
			for i := 0; i < len(clust.blips)-1; i++ {
				for j := i + 1; j < len(clust.blips); j++ {
					if clust.blips[i].dist > clust.blips[j].dist {
						clust.blips[i], clust.blips[j] = clust.blips[j], clust.blips[i]
					}
				}
			}

			// Calculate vertical offset for each blip
			stackStartY := clust.centerY - float64(len(clust.blips)-1)*radarStackSpacing*0.5
			for idx, b := range clust.blips {
				offsetY := float64(idx) * radarStackSpacing
				dotY := center.y + stackStartY + offsetY

				// For stacked dots, keep X at cluster center, adjust Y
				dotX := center.x + clust.centerX

				// Draw indicators first (so they appear behind the dot) - only for ships, not rocks
				if !b.isRock {
					if ship := g.GetShip(b.entityID); ship != nil {
						g.drawRadarIndicators(screen, ship, b.entityID, dotX, dotY, b.blipColor, player, scale, radarRadius, center)
					}
				}

				// Draw dot
				drawCircle(screen, dotX, dotY, radarBlipSize, b.blipColor)

				// Stack labels vertically as well, positioned relative to stacked dot
				var labelX, labelY float64
				if b.isOffRadar {
					// For off-radar blips, position label outward from the stacked dot
					labelX = dotX + b.dirX*radarOffRadarDist
					labelY = dotY + b.dirY*radarOffRadarDist
					// Clamp to radar bounds
					minX := center.x - radarRadius + 6
					maxX := center.x + radarRadius - 32
					minY := center.y - radarRadius + 6
					maxY := center.y + radarRadius - 12
					if labelX < minX {
						labelX = minX
					}
					if labelX > maxX {
						labelX = maxX
					}
					if labelY < minY {
						labelY = minY
					}
					if labelY > maxY {
						labelY = maxY
					}
				} else {
					// For on-radar blips, position label to the right and above the dot
					labelX = dotX + radarLabelOffsetX
					labelY = dotY - radarLabelOffsetY
				}
				ebitenutil.DebugPrintAt(screen, b.label, int(labelX), int(labelY))

				// Draw state label below distance - only for ships, not rocks
				if !b.isRock {
					state := g.getNPCState(b.entityID)
					stateLabel := g.getNPCStateString(state)
					ebitenutil.DebugPrintAt(screen, stateLabel, int(labelX), int(labelY)+12)
				}
			}
		}
	}
}

// drawRadarIndicators draws facing direction, engine burn, and speed vector indicators for an enemy on the radar
func (g *Game) drawRadarIndicators(screen *ebiten.Image, enemy *Ship, entityID EntityID, baseX, baseY float64, blipColor color.NRGBA, player *Ship, scale float64, radarRadius float64, radarCenter vec2) {
	// Facing direction triangle (smaller version of ship triangle)
	renderAngle := enemy.angle - player.angle
	// Triangle points in local space (nose up) - scaled down for radar
	nose := rotatePoint(vec2{0, -6}, renderAngle)
	left := rotatePoint(vec2{-4, 4}, renderAngle)
	right := rotatePoint(vec2{4, 4}, renderAngle)

	nose.x += baseX
	nose.y += baseY
	left.x += baseX
	left.y += baseY
	right.x += baseX
	right.y += baseY

	ebitenutil.DrawLine(screen, nose.x, nose.y, left.x, left.y, blipColor)
	ebitenutil.DrawLine(screen, left.x, left.y, right.x, right.y, blipColor)
	ebitenutil.DrawLine(screen, right.x, right.y, nose.x, nose.y, blipColor)

	// Calculate facing direction for engine burn indicator
	facingDir := rotatePoint(vec2{math.Sin(enemy.angle), -math.Cos(enemy.angle)}, -player.angle)

	// Main engine burn indicator (short flame behind when thrusting)
	if enemy.thrustThisFrame {
		flameLen := 8.0
		ebitenutil.DrawLine(screen, baseX, baseY, baseX-facingDir.x*flameLen, baseY-facingDir.y*flameLen, colorRadarFlame)
	}

	// Draw predictive trail instead of speed vector
	if npcInput, hasInput := g.npcInputs[entityID]; hasInput {
		predictedPositions := g.predictFuturePath(enemy, npcInput)
		g.drawPredictiveTrailInRadar(screen, predictedPositions, enemy, player, radarCenter, scale, radarRadius, colorRadarSpeedVector)
	}
}
