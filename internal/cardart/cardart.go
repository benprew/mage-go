// Package cardart generates unique deterministic pixel art for Magic: The Gathering cards.
// Each card gets a 32x32 image composed of layers: background (from color identity),
// silhouette (from card type), and a hash-based pattern overlay (from card name).
package cardart

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
	for y := 0; y < Size; y++ {
		for x := 0; x < Size; x++ {
			img.Set(x, y, bg(x, y))
		}
	}

	// Layer 2: frame border (2px)
	frameColor := frameColorFor(input.Colors)
	drawFrame(img, frameColor)

	// Layer 3: silhouette
	silhouette := getSilhouette(input.Type)
	silColor := silhouetteColor(input.Colors)

	// Layer 4: hash-based pattern overlay on silhouette
	pattern := generatePattern(hash)

	// Compose silhouette + pattern in the inner 24x24 region (offset 4,4)
	patternColor := patternColorFor(input.Colors)
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			if silhouette[y][x] {
				px, py := x+4, y+4
				if pattern[y][x] {
					img.Set(px, py, patternColor)
				} else {
					img.Set(px, py, silColor)
				}
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

func silhouetteColor(colors []Color) color.RGBA {
	if len(colors) == 0 {
		return color.RGBA{60, 60, 60, 255}
	}
	switch colors[0] {
	case ColorWhite:
		return color.RGBA{180, 170, 140, 255}
	case ColorBlue:
		return color.RGBA{60, 90, 170, 255}
	case ColorBlack:
		return color.RGBA{70, 50, 80, 255}
	case ColorRed:
		return color.RGBA{170, 60, 40, 255}
	case ColorGreen:
		return color.RGBA{50, 120, 55, 255}
	default:
		return color.RGBA{90, 90, 90, 255}
	}
}

func patternColorFor(colors []Color) color.RGBA {
	if len(colors) == 0 {
		return color.RGBA{140, 140, 140, 255}
	}
	switch colors[0] {
	case ColorWhite:
		return color.RGBA{255, 248, 220, 255}
	case ColorBlue:
		return color.RGBA{100, 150, 230, 255}
	case ColorBlack:
		return color.RGBA{120, 90, 140, 255}
	case ColorRed:
		return color.RGBA{230, 120, 80, 255}
	case ColorGreen:
		return color.RGBA{100, 190, 100, 255}
	default:
		return color.RGBA{170, 170, 170, 255}
	}
}

func drawFrame(img *image.RGBA, c color.RGBA) {
	for i := 0; i < Size; i++ {
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
func generatePattern(hash [32]byte) [24][24]bool {
	var pat [24][24]bool
	// We need 24*12 = 288 bits. We have 256 bits in hash.
	// Extend with a second round.
	hash2 := sha256.Sum256(hash[:])
	bits := append(hash[:], hash2[:]...)

	bitIdx := 0
	for y := 0; y < 24; y++ {
		for x := 0; x < 12; x++ {
			byteIdx := bitIdx / 8
			bitOff := uint(bitIdx % 8)
			if byteIdx < len(bits) && (bits[byteIdx]>>bitOff)&1 == 1 {
				pat[y][x] = true
				pat[y][23-x] = true // mirror
			}
			bitIdx++
		}
	}
	return pat
}

// drawOutline draws a 1px dark outline around silhouette edges for definition.
func drawOutline(img *image.RGBA, sil [24][24]bool, c color.RGBA) {
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
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
