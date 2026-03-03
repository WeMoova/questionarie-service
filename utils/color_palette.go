package utils

import (
	"fmt"
	"math"
	"strings"
)

const defaultPrimaryColor = "#6EC7E8"

// HexToHSL converts a hex color string (#RRGGBB) to HSL values.
func HexToHSL(hex string) (h, s, l float64) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0.5
	}

	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)

	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	l = (max + min) / 2

	if delta == 0 {
		return 0, 0, l
	}

	if l < 0.5 {
		s = delta / (max + min)
	} else {
		s = delta / (2.0 - max - min)
	}

	switch max {
	case rf:
		h = (gf - bf) / delta
		if gf < bf {
			h += 6
		}
	case gf:
		h = (bf-rf)/delta + 2
	case bf:
		h = (rf-gf)/delta + 4
	}
	h *= 60

	return h, s, l
}

// HSLToHex converts HSL values to a hex color string (#RRGGBB).
func HSLToHex(h, s, l float64) string {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}

	var r, g, b float64
	if s == 0 {
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		r = hueToRGB(p, q, h/360+1.0/3.0)
		g = hueToRGB(p, q, h/360)
		b = hueToRGB(p, q, h/360-1.0/3.0)
	}

	return fmt.Sprintf("#%02x%02x%02x",
		uint8(math.Round(r*255)),
		uint8(math.Round(g*255)),
		uint8(math.Round(b*255)),
	)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}

func clampL(l float64) float64 {
	if l < 0.1 {
		return 0.1
	}
	if l > 0.95 {
		return 0.95
	}
	return l
}

// GenerateLevelColors generates threshold level colors from a primary hex color.
// bajo → lighter, medio → near primary, alto → darker.
func GenerateLevelColors(primaryHex string) map[string]string {
	if primaryHex == "" {
		primaryHex = defaultPrimaryColor
	}
	h, s, l := HexToHSL(primaryHex)

	return map[string]string{
		"bajo":     HSLToHex(h, s, clampL(l+0.25)),
		"medio":    HSLToHex(h, s, clampL(l)),
		"alto":     HSLToHex(h, s, clampL(l-0.25)),
		"muy_bajo": HSLToHex(h, s, clampL(l+0.35)),
		"muy_alto": HSLToHex(h, s, clampL(l-0.35)),
	}
}

// GenerateSeverityColors generates risk profile severity colors from a primary hex color.
// low → lightest, medium → light, high → primary, critical → darkest.
func GenerateSeverityColors(primaryHex string) map[string]string {
	if primaryHex == "" {
		primaryHex = defaultPrimaryColor
	}
	h, s, l := HexToHSL(primaryHex)

	return map[string]string{
		"low":      HSLToHex(h, s, clampL(l+0.30)),
		"medium":   HSLToHex(h, s, clampL(l+0.10)),
		"high":     HSLToHex(h, s, clampL(l)),
		"critical": HSLToHex(h, s, clampL(l-0.20)),
		"info":     HSLToHex(h, s, clampL(l+0.30)),
		"warning":  HSLToHex(h, s, clampL(l+0.10)),
	}
}
