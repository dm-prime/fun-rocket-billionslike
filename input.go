package main

import "github.com/hajimehoshi/ebiten/v2"

// handleInput processes input events like fullscreen toggle and restart
func (g *Game) handleInput() {
	// Handle R key to restart (only when game over)
	if g.gameOver {
		restartPressed := ebiten.IsKeyPressed(ebiten.KeyR)
		if restartPressed && !g.prevRestartKey {
			g.restart()
		}
		g.prevRestartKey = restartPressed
		return // Don't process other input when game over
	}

	// Handle Alt+Enter to toggle fullscreen
	altPressed := ebiten.IsKeyPressed(ebiten.KeyAlt) || ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight)
	enterPressed := ebiten.IsKeyPressed(ebiten.KeyEnter)
	altEnterPressed := altPressed && enterPressed

	if altEnterPressed && !g.prevAltEnter {
		// Toggle fullscreen
		isCurrentlyFullscreen := ebiten.IsFullscreen()
		ebiten.SetFullscreen(!isCurrentlyFullscreen)

		// Update screen size when toggling
		if !isCurrentlyFullscreen {
			// Going to fullscreen - use monitor size
			monitorWidth, monitorHeight := ebiten.ScreenSizeInFullscreen()
			screenWidth = monitorWidth
			screenHeight = monitorHeight
		} else {
			// Going to windowed - use 90% of monitor size for a reasonable window
			monitorWidth, monitorHeight := ebiten.ScreenSizeInFullscreen()
			screenWidth = int(float64(monitorWidth) * windowedSizeRatio)
			screenHeight = int(float64(monitorHeight) * windowedSizeRatio)
		}
		ebiten.SetWindowSize(screenWidth, screenHeight)
	}
	g.prevAltEnter = altEnterPressed
}

// handlePlayerShooting handles spacebar input for player shooting
func (g *Game) handlePlayerShooting(player *Ship) {
	spacePressed := ebiten.IsKeyPressed(ebiten.KeySpace)

	// Fire on key press (not while held)
	if spacePressed && !g.prevSpaceKey {
		g.firePlayerTurrets(player)
	}

	g.prevSpaceKey = spacePressed
}

// getPlayerInput reads keyboard input and returns ShipInput for the player
func getPlayerInput() ShipInput {
	return ShipInput{
		TurnLeft:      ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA),
		TurnRight:     ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD),
		ThrustForward: ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW),
		ReverseThrust: ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS),
	}
}
