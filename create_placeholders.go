package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func createPlaceholderImage(filename string, width, height int, clr color.Color) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, clr)
		}
	}
	
	// Draw a simple shape (triangle for ship)
	centerX := width / 2
	centerY := height / 2
	
	// Draw outline in darker color
	darkClr := color.RGBA{0, 0, 0, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Simple triangle shape
			relX := float64(x - centerX)
			relY := float64(y - centerY)
			
			// Triangle bounds
			if relY > -float64(height)/3 && relY < float64(height)/3 {
				edgeX := float64(width)/4 * (1 - abs(relY)/(float64(height)/3))
				if abs(relX) < edgeX {
					img.Set(x, y, clr)
				} else if abs(relX) < edgeX+1 {
					img.Set(x, y, darkClr)
				}
			}
		}
	}
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	return png.Encode(file, img)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func main() {
	// Create player image (blue/white)
	createPlaceholderImage("assets/player.png", 32, 32, color.RGBA{100, 150, 255, 255})
	
	// Create enemy image (red)
	createPlaceholderImage("assets/enemy.png", 32, 32, color.RGBA{255, 100, 100, 255})
	
	// Create rocket image (orange/yellow)
	createPlaceholderImage("assets/rocket.png", 16, 16, color.RGBA{255, 200, 0, 255})
	
	// Create rock image (brown/gray)
	createPlaceholderImage("assets/rock.png", 24, 24, color.RGBA{120, 100, 80, 255})
}

