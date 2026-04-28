// Package cardart generates unique deterministic pixel art for Magic: The Gathering cards.
// Each card gets a 32x32 image composed of layers: background (from color identity),
// silhouette (from card type), and a hash-based pattern overlay (from card name).
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"image"
	"image/color"
	"math"
)

// CardType determines the silhouette shape.
type CardType int

const (
	TypeCreature CardType = iota
	TypeArtifact
	TypeEnchantment
	TypeInstant
	TypeSorcery
	TypeLand
)

// Color represents a Magic color for palette selection.
type Color int

const (
	ColorWhite Color = iota
	ColorBlue
	ColorBlack
	ColorRed
	ColorGreen
	ColorColorless
)

// CardInput holds the properties used to generate a card image.
type CardInput struct {
	Name   string
	Colors []Color
	Type   CardType
}

const Size = 32

// Generate produces a 32x32 pixel art image for the given card input.
func Generate(input CardInput) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, Size, Size))
	hash := sha256.Sum256([]byte(input.Name))

	// Layer 1: background
	bg := backgroundGradient(input.Colors, hash)
	for y := range Size {
		for x := range Size {
			img.Set(x, y, bg(x, y))
		}
	}

	// Layer 2: frame border (2px)
	frameColor := frameColorFor(input.Colors)
	drawFrame(img, frameColor)

	// Layer 3: silhouette
	silhouette := getSilhouette(input.Type)

	// Layer 4: hash-based pattern overlay on silhouette
	pattern := generatePattern(hash)

	// Compose silhouette + pattern in the inner 24x24 region (offset 4,4)
	// Pattern has 4 intensity levels (0-3) for richer visual detail
	patColors := patternPalette(input.Colors)
	for y := range 24 {
		for x := range 24 {
			if silhouette[y][x] {
				px, py := x+4, y+4
				img.Set(px, py, patColors[pattern[y][x]])
			}
		}
	}

	// Layer 5: silhouette outline for definition
	outlineColor := color.RGBA{0, 0, 0, 180}
	drawOutline(img, silhouette, outlineColor)

	return img
}

// backgroundGradient returns a function that gives the bg color at (x,y).
func backgroundGradient(colors []Color, hash [32]byte) func(x, y int) color.RGBA {
	if len(colors) == 0 {
		colors = []Color{ColorColorless}
	}

	palettes := make([]color.RGBA, len(colors))
	for i, c := range colors {
		palettes[i] = bgPalette(c)
	}

	// Use hash bytes for subtle noise
	return func(x, y int) color.RGBA {
		// Blend colors based on position
		t := float64(x+y) / float64(2*Size)
		var r, g, b float64
		if len(palettes) == 1 {
			base := palettes[0]
			r, g, b = float64(base.R), float64(base.G), float64(base.B)
		} else {
			idx := t * float64(len(palettes)-1)
			lo := int(math.Floor(idx))
			hi := lo + 1
			if hi >= len(palettes) {
				hi = len(palettes) - 1
				lo = hi - 1
			}
			f := idx - float64(lo)
			r = lerp(float64(palettes[lo].R), float64(palettes[hi].R), f)
			g = lerp(float64(palettes[lo].G), float64(palettes[hi].G), f)
			b = lerp(float64(palettes[lo].B), float64(palettes[hi].B), f)
		}

		// Subtle hash-based noise for texture
		noiseIdx := (y*Size + x) % 32
		noise := float64(hash[noiseIdx]%16) - 8
		r = clampF(r+noise, 0, 255)
		g = clampF(g+noise, 0, 255)
		b = clampF(b+noise, 0, 255)

		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}
}

func bgPalette(c Color) color.RGBA {
	switch c {
	case ColorWhite:
		return color.RGBA{240, 230, 200, 255} // warm parchment
	case ColorBlue:
		return color.RGBA{40, 60, 130, 255} // deep blue
	case ColorBlack:
		return color.RGBA{50, 35, 55, 255} // dark purple-black
	case ColorRed:
		return color.RGBA{140, 30, 30, 255} // crimson
	case ColorGreen:
		return color.RGBA{30, 90, 40, 255} // forest green
	default:
		return color.RGBA{120, 115, 110, 255} // stone grey
	}
}

func frameColorFor(colors []Color) color.RGBA {
	if len(colors) == 0 {
		return color.RGBA{80, 80, 80, 255}
	}
	switch colors[0] {
	case ColorWhite:
		return color.RGBA{200, 190, 150, 255}
	case ColorBlue:
		return color.RGBA{20, 40, 100, 255}
	case ColorBlack:
		return color.RGBA{30, 20, 35, 255}
	case ColorRed:
		return color.RGBA{100, 20, 20, 255}
	case ColorGreen:
		return color.RGBA{20, 60, 25, 255}
	default:
		return color.RGBA{70, 70, 70, 255}
	}
}

// patternPalette returns 4 colors (indexed 0-3) for the pattern intensity levels.
// Level 0 is darkest (shadow), level 3 is brightest (highlight).
func patternPalette(colors []Color) [4]color.RGBA {
	if len(colors) == 0 {
		return [4]color.RGBA{
			{50, 50, 50, 255},
			{80, 80, 80, 255},
			{130, 130, 130, 255},
			{175, 175, 175, 255},
		}
	}
	switch colors[0] {
	case ColorWhite:
		return [4]color.RGBA{
			{170, 160, 130, 255},
			{200, 190, 160, 255},
			{230, 220, 190, 255},
			{255, 248, 220, 255},
		}
	case ColorBlue:
		return [4]color.RGBA{
			{30, 50, 120, 255},
			{50, 80, 160, 255},
			{80, 120, 200, 255},
			{120, 170, 245, 255},
		}
	case ColorBlack:
		return [4]color.RGBA{
			{35, 25, 40, 255},
			{60, 45, 70, 255},
			{90, 70, 110, 255},
			{130, 100, 150, 255},
		}
	case ColorRed:
		return [4]color.RGBA{
			{100, 25, 15, 255},
			{150, 50, 30, 255},
			{200, 85, 55, 255},
			{240, 130, 90, 255},
		}
	case ColorGreen:
		return [4]color.RGBA{
			{25, 70, 30, 255},
			{45, 110, 50, 255},
			{75, 155, 75, 255},
			{115, 200, 110, 255},
		}
	default:
		return [4]color.RGBA{
			{50, 50, 50, 255},
			{80, 80, 80, 255},
			{130, 130, 130, 255},
			{175, 175, 175, 255},
		}
	}
}

func drawFrame(img *image.RGBA, c color.RGBA) {
	for i := range Size {
		// Top and bottom, 2px thick
		img.Set(i, 0, c)
		img.Set(i, 1, c)
		img.Set(i, Size-1, c)
		img.Set(i, Size-2, c)
		// Left and right, 2px thick
		img.Set(0, i, c)
		img.Set(1, i, c)
		img.Set(Size-1, i, c)
		img.Set(Size-2, i, c)
	}
}

// generatePattern creates a 24x24 vertically-symmetric pattern from a hash.
// Uses the left 12 columns, mirrored to the right, for an identicon-like feel.
// Returns intensity levels 0-3 for richer visual detail.
func generatePattern(hash [32]byte) [24][24]uint8 {
	var pat [24][24]uint8
	// We need 24*12*2 = 576 bits for 2-bit values. Extend hash.
	hash2 := sha256.Sum256(hash[:])
	hash3 := sha256.Sum256(hash2[:])
	bits := append(hash[:], hash2[:]...)
	bits = append(bits, hash3[:]...)

	bitIdx := 0
	for y := range 24 {
		for x := range 12 {
			// Read 2 bits for 4 intensity levels
			byteIdx := bitIdx / 8
			bitOff := uint(bitIdx % 8)
			var val uint8
			if byteIdx < len(bits) {
				val = (bits[byteIdx] >> bitOff) & 0x3
			}
			pat[y][x] = val
			pat[y][23-x] = val // mirror
			bitIdx += 2
		}
	}
	return pat
}

// drawOutline draws a 1px dark outline around silhouette edges for definition.
func drawOutline(img *image.RGBA, sil [24][24]bool, c color.RGBA) {
	for y := range 24 {
		for x := range 24 {
			if !sil[y][x] {
				continue
			}
			// Check if this pixel borders a non-silhouette pixel
			isEdge := false
			for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || nx >= 24 || ny < 0 || ny >= 24 || !sil[ny][nx] {
					isEdge = true
					break
				}
			}
			if isEdge {
				img.Set(x+4, y+4, c)
			}
		}
	}
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// HashToSeed converts first 8 bytes of a hash to a uint64 for deterministic RNG.
func HashToSeed(hash [32]byte) uint64 {
	return binary.LittleEndian.Uint64(hash[:8])
}
