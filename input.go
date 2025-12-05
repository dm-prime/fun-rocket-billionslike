package main

import "github.com/hajimehoshi/ebiten/v2"

// handleInput processes input events like fullscreen toggle
func (g *Game) handleInput() {
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

// getPlayerInput reads keyboard input and returns ShipInput for the player
func getPlayerInput() ShipInput {
	return ShipInput{
		TurnLeft:       ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA),
		TurnRight:      ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD),
		ThrustForward:  ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW),
		RetrogradeBurn: ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS),
	}
}
