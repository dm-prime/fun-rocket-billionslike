package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"

	"billionslike3/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Start pprof HTTP server in a goroutine for profiling
	go func() {
		log.Println("Starting pprof server on http://localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	config := game.DefaultConfig()
	g := game.NewGame(config)

	ebiten.SetWindowSize(config.ScreenWidth, config.ScreenHeight)
	ebiten.SetWindowTitle("Space Shooter")
	ebiten.SetWindowResizable(true)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
