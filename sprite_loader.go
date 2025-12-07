package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

//go:embed assets/ship.svg
var shipSVGData []byte

var (
	playerShipSprite *ebiten.Image
	shipSpriteWidth  = 48
	shipSpriteHeight = 48
)

// initSprites loads and converts SVG assets to PNG sprites
func initSprites() error {
	// Convert SVG to PNG
	shipPNG, err := svgToPNG(shipSVGData, shipSpriteWidth, shipSpriteHeight)
	if err != nil {
		return err
	}

	// Convert to ebiten image
	playerShipSprite = ebiten.NewImageFromImage(shipPNG)

	// Optionally save PNG for debugging
	if os.Getenv("DEBUG_SPRITES") == "1" {
		saveDebugPNG(shipPNG, "debug_ship.png")
	}

	return nil
}

// svgToPNG converts SVG data to a PNG image
func svgToPNG(svgData []byte, width, height int) (image.Image, error) {
	// Parse SVG
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgData))
	if err != nil {
		return nil, err
	}

	// Set the target size
	icon.SetTarget(0, 0, float64(width), float64(height))

	// Create RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create scanner and rasterize
	scanner := rasterx.NewScannerGV(width, height, img, img.Bounds())
	raster := rasterx.NewDasher(width, height, scanner)

	// Render SVG to image
	icon.Draw(raster, 1.0)

	return img, nil
}

// saveDebugPNG saves a PNG image for debugging purposes
func saveDebugPNG(img image.Image, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Printf("Failed to create debug PNG: %v", err)
		return
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		log.Printf("Failed to encode debug PNG: %v", err)
	}
}

