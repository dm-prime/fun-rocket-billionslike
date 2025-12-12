package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"billionslike3/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	// Parse command line flags
	convexURL := flag.String("convex-url", "", "Convex deployment URL (or set CONVEX_URL env var)")
	testMode := flag.Bool("test", false, "Run in test mode with example scripts")
	flag.Parse()

	// Get Convex URL from flag or environment
	url := *convexURL
	if url == "" {
		url = os.Getenv("CONVEX_URL")
	}

	if url == "" && !*testMode {
		fmt.Println("Convex Mode - Space Shooter with Script-Driven AI")
		fmt.Println("")
		fmt.Println("Usage:")
		fmt.Println("  Set CONVEX_URL environment variable or use -convex-url flag")
		fmt.Println("  Or use -test flag to run with example scripts")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  cmd/convex/main.go -convex-url https://your-deployment.convex.cloud")
		fmt.Println("  CONVEX_URL=https://your-deployment.convex.cloud cmd/convex/main.go")
		fmt.Println("  cmd/convex/main.go -test")
		fmt.Println("")
		fmt.Println("Running in test mode...")
		*testMode = true
	}

	// Set up runtime
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Printf("Starting Convex Mode with GOMAXPROCS=%d\n", runtime.GOMAXPROCS(0))

	// Create game config
	config := game.DefaultConfig()

	// Create Convex game
	g, err := game.NewConvexGame(config, url)
	if err != nil {
		log.Fatalf("Failed to create Convex game: %v", err)
	}
	defer g.Close()

	// If Convex URL is provided, try to fetch scripts
	if url != "" {
		log.Printf("Connecting to Convex at %s...\n", url)
		if err := g.RefreshScripts(); err != nil {
			log.Printf("Warning: Failed to fetch scripts: %v\n", err)
			log.Println("Running with local scripts only...")
		} else {
			log.Println("Successfully connected to Convex!")
		}
	}

	// In test mode, add example scripts
	if *testMode {
		log.Println("Loading example scripts...")

		// Add chase script
		if err := g.AddScript("chase", game.GetExampleScript()); err != nil {
			log.Printf("Failed to add chase script: %v\n", err)
		}

		// Add circle script
		if err := g.AddScript("circle", game.GetCircleScript()); err != nil {
			log.Printf("Failed to add circle script: %v\n", err)
		}

		// Spawn some test enemies
		go func() {
			time.Sleep(2 * time.Second) // Wait for game to initialize
			log.Println("Spawning test enemies...")

			// Spawn enemies with different scripts
			for i := 0; i < 3; i++ {
				g.SpawnScriptedEnemy("chase")
			}
			for i := 0; i < 2; i++ {
				g.SpawnScriptedEnemy("circle")
			}
		}()
	}

	// Set up window
	ebiten.SetWindowSize(config.ScreenWidth, config.ScreenHeight)
	ebiten.SetWindowTitle("Space Shooter - Convex Mode")
	ebiten.SetWindowResizable(true)

	log.Println("Starting game loop...")

	// Run game
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
