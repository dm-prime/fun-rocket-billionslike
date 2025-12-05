package main

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// drawOffscreenIndicators draws edge-of-screen markers for enemies that are not visible.
func (g *Game) drawOffscreenIndicators(screen *ebiten.Image, player *Ship) {
	screenCenter := vec2{float64(screenWidth) * 0.5, float64(screenHeight) * 0.5}
	minX := indicatorMargin
	maxX := float64(screenWidth) - indicatorMargin
	minY := indicatorMargin
	maxY := float64(screenHeight) - indicatorMargin

	type cornerStat struct {
		count   int
		minDist float64
		dir     vec2
		pos     vec2
		clr     color.Color
	}
	corners := map[string]*cornerStat{}

	drawIndicator := func(pos vec2, dir vec2, dist float64, count int, clr color.Color) {
		tipX := pos.x + dir.x*indicatorArrowLen*0.6
		tipY := pos.y + dir.y*indicatorArrowLen*0.6
		tailX := pos.x - dir.x*indicatorArrowLen*0.4
		tailY := pos.y - dir.y*indicatorArrowLen*0.4
		ebitenutil.DrawLine(screen, tailX, tailY, tipX, tipY, clr)

		wingAngle := math.Pi / 6
		sinA := math.Sin(wingAngle)
		cosA := math.Cos(wingAngle)
		leftWing := vec2{
			x: dir.x*cosA - dir.y*sinA,
			y: dir.x*sinA + dir.y*cosA,
		}
		rightWing := vec2{
			x: dir.x*cosA + dir.y*sinA,
			y: -dir.x*sinA + dir.y*cosA,
		}
		wingLen := indicatorArrowLen * 0.5
		ebitenutil.DrawLine(screen, tipX, tipY, tipX-leftWing.x*wingLen, tipY-leftWing.y*wingLen, clr)
		ebitenutil.DrawLine(screen, tipX, tipY, tipX-rightWing.x*wingLen, tipY-rightWing.y*wingLen, clr)

		label := fmt.Sprintf("%.0f", dist)
		if count > 1 {
			label = fmt.Sprintf("%.0f (x%d)", dist, count)
		}
		labelX := pos.x + indicatorLabelX
		labelY := pos.y - indicatorLabelY
		maxLabelX := float64(screenWidth) - hudLabelMarginX
		if labelX > maxLabelX {
			labelX = maxLabelX
		}
		if labelX < 4 {
			labelX = 4
		}
		if labelY < 4 {
			labelY = 4
		}
		if labelY > float64(screenHeight)-hudLabelMarginY {
			labelY = float64(screenHeight) - hudLabelMarginY
		}
		ebitenutil.DebugPrintAt(screen, label, int(labelX), int(labelY))
	}

	for i := range g.ships {
		if i == g.playerIndex {
			continue
		}
		enemy := &g.ships[i]
		indicatorColor := g.colorForFaction(enemy.faction)

		dx := enemy.pos.x - player.pos.x
		dy := enemy.pos.y - player.pos.y
		dist := math.Hypot(dx, dy)
		if dist < 1 {
			continue
		}

		// Rotate world around player so player stays upright.
		rot := rotatePoint(vec2{dx, dy}, -player.angle)
		screenX := screenCenter.x + rot.x
		screenY := screenCenter.y + rot.y

		// If on-screen, skip indicator.
		if screenX >= 0 && screenX <= float64(screenWidth) && screenY >= 0 && screenY <= float64(screenHeight) {
			continue
		}

		// Clamp to edge with margin.
		clampedX := math.Min(math.Max(screenX, minX), maxX)
		clampedY := math.Min(math.Max(screenY, minY), maxY)

		dirX := rot.x / math.Hypot(rot.x, rot.y)
		dirY := rot.y / math.Hypot(rot.x, rot.y)

		isCorner := (clampedX == minX || clampedX == maxX) && (clampedY == minY || clampedY == maxY)
		if isCorner {
			key := fmt.Sprintf("%t-%t", clampedX == minX, clampedY == minY) // left/right - top/bottom
			if stat, ok := corners[key]; ok {
				stat.count++
				if dist < stat.minDist {
					stat.minDist = dist
					stat.dir = vec2{dirX, dirY}
					stat.pos = vec2{clampedX, clampedY}
					stat.clr = indicatorColor
				}
			} else {
				corners[key] = &cornerStat{
					count:   1,
					minDist: dist,
					dir:     vec2{dirX, dirY},
					pos:     vec2{clampedX, clampedY},
					clr:     indicatorColor,
				}
			}
			continue
		}

		drawIndicator(vec2{clampedX, clampedY}, vec2{dirX, dirY}, dist, 1, indicatorColor)
	}

	for _, stat := range corners {
		drawIndicator(stat.pos, stat.dir, stat.minDist, stat.count, stat.clr)
	}
}

// drawHUD draws the heads-up display with player stats
func (g *Game) drawHUD(screen *ebiten.Image, player *Ship) {
	retroStatus := ""
	if player.retrogradeMode {
		speed := math.Hypot(player.vel.x, player.vel.y)
		targetAngle := math.Atan2(-player.vel.x, player.vel.y)
		angleDiff := math.Abs(normalizeAngle(targetAngle-player.angle)) * 180 / math.Pi
		if angleDiff > 20 {
			retroStatus = fmt.Sprintf(" | RETROGRADE: TURNING (%.0f° off)", angleDiff)
		} else {
			retroStatus = fmt.Sprintf(" | RETROGRADE: BURNING (speed: %.1f)", speed)
		}
	}
	hud := fmt.Sprintf("Speed: %0.1f | Angular: %0.2f rad/s%s",
		math.Hypot(player.vel.x, player.vel.y), player.angularVel, retroStatus)
	ebitenutil.DebugPrint(screen, hud)
}
