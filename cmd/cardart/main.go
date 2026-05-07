// Command cardart generates unique pixel art images for Magic: The Gathering cards.
//
// Usage:
//
//	cardart -name "Lightning Bolt" -color R -type instant
//	cardart -name "Shivan Dragon" -color R -type creature -o dragon.png
//	cardart -name "Shivan Dragon" -color R -type creature -o dragon.svg
//	cardart -name "Counterspell" -color U -type instant -ansi
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	os.Exit(run())
}

func run() int {
	name := flag.String("name", "", "card name (required)")
	colorStr := flag.String("color", "", "color identity: W, U, B, R, G, or combinations like WU, BRG (empty = colorless)")
	typeStr := flag.String("type", "creature", "card type: creature, artifact, enchantment, instant, sorcery, land")
	output := flag.String("o", "", "output file path; format detected from extension (.png or .svg, default: .png)")
	svgScale := flag.Int("scale", 8, "SVG pixel scale factor (each game pixel = NxN SVG pixels)")
	ansi := flag.Bool("ansi", false, "force ANSI terminal output even when -o is set")
	flag.Parse()

	if *name == "" {
		fmt.Fprintln(os.Stderr, "error: -name is required")
		flag.Usage()
		return 1
	}

	input := CardInput{
		Name:   *name,
		Colors: parseColors(*colorStr),
		Type:   parseType(*typeStr),
	}

	img := Generate(input)

	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating file: %v\n", err)
			return 1
		}
		defer f.Close()

		isSVG := strings.HasSuffix(strings.ToLower(*output), ".svg")
		if isSVG {
			err = WriteSVG(f, img, *svgScale)
		} else {
			err = WritePNG(f, img)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing file: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", *output)
	}

	if *output == "" || *ansi {
		fmt.Print(RenderANSI(img))
	}
	return 0
}

func parseColors(s string) []Color {
	if s == "" {
		return nil
	}
	var colors []Color
	for _, ch := range strings.ToUpper(s) {
		switch ch {
		case 'W':
			colors = append(colors, ColorWhite)
		case 'U':
			colors = append(colors, ColorBlue)
		case 'B':
			colors = append(colors, ColorBlack)
		case 'R':
			colors = append(colors, ColorRed)
		case 'G':
			colors = append(colors, ColorGreen)
		}
	}
	return colors
}

func parseType(s string) CardType {
	switch strings.ToLower(s) {
	case "creature":
		return TypeCreature
	case "artifact":
		return TypeArtifact
	case "enchantment":
		return TypeEnchantment
	case "instant":
		return TypeInstant
	case "sorcery":
		return TypeSorcery
	case "land":
		return TypeLand
	default:
		return TypeCreature
	}
}
