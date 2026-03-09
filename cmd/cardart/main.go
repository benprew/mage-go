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

	"github.com/mage/mage/internal/cardart"
)

func main() {
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
		os.Exit(1)
	}

	input := cardart.CardInput{
		Name:   *name,
		Colors: parseColors(*colorStr),
		Type:   parseType(*typeStr),
	}

	img := cardart.Generate(input)

	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		isSVG := strings.HasSuffix(strings.ToLower(*output), ".svg")
		if isSVG {
			err = cardart.WriteSVG(f, img, *svgScale)
		} else {
			err = cardart.WritePNG(f, img)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", *output)
	}

	if *output == "" || *ansi {
		fmt.Print(cardart.RenderANSI(img))
	}
}

func parseColors(s string) []cardart.Color {
	if s == "" {
		return nil
	}
	var colors []cardart.Color
	for _, ch := range strings.ToUpper(s) {
		switch ch {
		case 'W':
			colors = append(colors, cardart.ColorWhite)
		case 'U':
			colors = append(colors, cardart.ColorBlue)
		case 'B':
			colors = append(colors, cardart.ColorBlack)
		case 'R':
			colors = append(colors, cardart.ColorRed)
		case 'G':
			colors = append(colors, cardart.ColorGreen)
		}
	}
	return colors
}

func parseType(s string) cardart.CardType {
	switch strings.ToLower(s) {
	case "creature":
		return cardart.TypeCreature
	case "artifact":
		return cardart.TypeArtifact
	case "enchantment":
		return cardart.TypeEnchantment
	case "instant":
		return cardart.TypeInstant
	case "sorcery":
		return cardart.TypeSorcery
	case "land":
		return cardart.TypeLand
	default:
		return cardart.TypeCreature
	}
}
