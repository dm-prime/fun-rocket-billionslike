package main

import (
	"log"

	"billionslike3/game"

	"github.com/hajimehoshi/ebiten/v2"
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
