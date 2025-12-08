package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"billionslike3/game"
)

func main() {
	config := game.DefaultConfig()
	g := game.NewGame(config)

	ebiten.SetWindowSize(config.ScreenWidth, config.ScreenHeight)
	ebiten.SetWindowTitle("Space Shooter")
	ebiten.SetWindowResizable(true)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

