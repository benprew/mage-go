package cardart

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"strings"
)

// WritePNG writes the image as a PNG to the given writer.
func WritePNG(w io.Writer, img *image.RGBA) error {
	return png.Encode(w, img)
}

// WriteSVG writes the image as a scalable SVG to the given writer.
// Each pixel becomes a rect element. The SVG scales cleanly to any size.
func WriteSVG(w io.Writer, img *image.RGBA, scale int) error {
	if scale < 1 {
		scale = 8
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	svgW, svgH := width*scale, height*scale

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" shape-rendering="crispEdges">`, svgW, svgH)
	sb.WriteByte('\n')

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="%d" height="%d" fill="rgb(%d,%d,%d)"/>`,
				x*scale, y*scale, scale, scale, r>>8, g>>8, b>>8)
			sb.WriteByte('\n')
		}
	}

	sb.WriteString("</svg>\n")
	_, err := io.WriteString(w, sb.String())
	return err
}

// RenderANSI returns a string of ANSI-colored block characters representing the image.
// Each pixel becomes a half-block character. Two vertical pixels share one character cell
// using the upper-half-block (▀) with foreground=top pixel, background=bottom pixel.
func RenderANSI(img *image.RGBA) string {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	var sb strings.Builder
	for y := 0; y < h; y += 2 {
		for x := 0; x < w; x++ {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			if y+1 < h {
				r2, g2, b2, _ := img.At(x, y+1).RGBA()
				// Upper half = top pixel (foreground), lower half = bottom pixel (background)
				sb.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm▀",
					r1>>8, g1>>8, b1>>8,
					r2>>8, g2>>8, b2>>8))
			} else {
				// Odd height: just the top pixel
				sb.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm▀", r1>>8, g1>>8, b1>>8))
			}
		}
		sb.WriteString("\033[0m\n")
	}
	return sb.String()
}
