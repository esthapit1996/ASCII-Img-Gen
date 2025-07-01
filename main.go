package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	_ "image/gif"
	_ "image/jpeg"

	"github.com/nfnt/resize"
)

var asciiChars = "@$#%&MW8BZO0QLCJUYXzcvunxrjft/\\|()1{}[]?-_+~<>i!lI;:,.\"^`' "

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <image-path>")
	}

	filePath := os.Args[1]
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open image: %v", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatalf("Failed to decode image: %v", err)
	}

	// Get original bounds
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate scale preserving aspect ratio to fit within 164x164 box
	maxSize := 164
	var newWidth, newHeight uint

	if origWidth > origHeight {
		newWidth = uint(maxSize)
		newHeight = uint(origHeight * maxSize / origWidth)
	} else {
		newHeight = uint(maxSize)
		newWidth = uint(origWidth * maxSize / origHeight)
	}
	imgNoBg := replaceBackgroundWithGray(img)
	resized := resize.Resize(newWidth, newHeight, imgNoBg, resize.Lanczos3)

	// desiredWidth := 80 // smaller width for ASCII output
	// resized := resize.Resize(uint(desiredWidth), 0, img, resize.Lanczos3)

	asciiLines := getASCIIArt(resized)

	inputName := filepath.Base(filePath)
	asciiName := "ascii-" + inputName
	baseOutputPath := filepath.Join("output", asciiName)
	outputPath := getAvailableFilename(baseOutputPath)

	saveAsPNG(asciiLines, outputPath)
}

// Helper function to check if a pixel is close to white (background)
func isBackground(c color.Color) bool {
	r, g, b, _ := c.RGBA()

	// Convert from 16-bit (0-65535) to 8-bit (0-255)
	R := float64(r >> 8)
	G := float64(g >> 8)
	B := float64(b >> 8)

	// Threshold for "closeness" to white
	threshold := 240.0

	// Simple Euclidean distance from white
	dist := math.Sqrt((255-R)*(255-R) + (255-G)*(255-G) + (255-B)*(255-B))
	return dist < (255 - threshold)
}

// Replace background pixels with gray color in the image
func replaceBackgroundWithGray(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	grayColor := color.RGBA{128, 128, 128, 255} // medium gray

	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			if isBackground(originalColor) {
				rgba.Set(x, y, grayColor)
			} else {
				rgba.Set(x, y, originalColor)
			}
		}
	}
	return rgba
}

// getAvailableFilename checks if path exists, and if so, adds -1, -2 etc before the extension
func getAvailableFilename(basePath string) string {
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return basePath
	}

	ext := filepath.Ext(basePath)
	name := strings.TrimSuffix(filepath.Base(basePath), ext)
	dir := filepath.Dir(basePath)

	for i := 1; ; i++ {
		newName := fmt.Sprintf("%s-%d%s", name, i, ext)
		newPath := filepath.Join(dir, newName)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}
}

func getASCIIArt(img image.Image) []string {
	bounds := img.Bounds()
	lines := []string{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y += 2 {
		line := ""
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			gray := uint8((r + g + b) / 3 >> 8)
			char := mapBrightnessToChar(gray)
			line += string(char)
		}
		lines = append(lines, line)
	}
	return lines
}

func mapBrightnessToChar(brightness uint8) byte {
	scale := float64(brightness) / 255.0
	index := int(scale * float64(len(asciiChars)-1))
	return asciiChars[index]
}

func saveAsPNG(lines []string, outputPath string) {
	fontFace := basicfont.Face7x13
	charWidth := 7
	charHeight := 13

	imgWidth := len(lines[0]) * charWidth
	imgHeight := len(lines) * charHeight

	rgba := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	col := color.Black

	for i, line := range lines {
		for j, c := range line {
			d := &font.Drawer{
				Dst:  rgba,
				Src:  image.NewUniform(col),
				Face: fontFace,
				Dot: fixed.P(
					j*charWidth,
					(i+1)*charHeight,
				),
			}
			d.DrawString(string(c))
		}
	}

	// ✅ Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	// ✅ Save image to file
	outFile, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, rgba); err != nil {
		log.Fatalf("Failed to encode PNG: %v", err)
	}

	fmt.Println("✅ ASCII image saved to", outputPath)
}
