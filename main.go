package main

import (
	"log"
	"net/http"
	"os"
	"runtime"
	_ "net/http/pprof"

	"billionslike3/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Tune GC for better performance in games
	// Set GOGC to 100 (default) but ensure we're using the latest GC
	// For games, we want lower latency, so we can set a lower GOGC if needed
	// But for now, keep default and monitor
	
	// Set minimum number of OS threads to match CPU count for better parallelism
	// This helps with GC and game loop parallelism
	runtime.GOMAXPROCS(runtime.NumCPU())
	
	log.Printf("GC tuning: GOGC=%s, GOMAXPROCS=%d\n", 
		os.Getenv("GOGC"), runtime.GOMAXPROCS(0))
	
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
