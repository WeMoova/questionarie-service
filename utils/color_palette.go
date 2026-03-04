package utils

import (
	"fmt"
	"math"
	"strings"

	"questionarie-service/models"
)

const DefaultPrimaryColor = "#6EC7E8"

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
		primaryHex = DefaultPrimaryColor
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
		primaryHex = DefaultPrimaryColor
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

// GenerateDistinctHues generates N visually distinct colors from the primary.
// Uses a bounded hue spread (not full 360°) so all colors stay in the same
// color family, matching the frontend's dimensionBases() algorithm:
//   - 1-3 dims: ±15° hue
//   - 4-6 dims: ±25° hue + saturation shifts
//   - 7+  dims: ±35° hue + saturation + lightness shifts
func GenerateDistinctHues(primaryHex string, n int) []string {
	if primaryHex == "" {
		primaryHex = DefaultPrimaryColor
	}
	if n <= 0 {
		return nil
	}

	h, s, l := HexToHSL(primaryHex)

	// Ensure good saturation and mid-range lightness
	if s < 0.25 {
		s = 0.55
	}
	if l < 0.35 {
		l = 0.45
	} else if l > 0.60 {
		l = 0.50
	}

	if n == 1 {
		return []string{HSLToHex(h, s, l)}
	}

	hueSpread := 15.0
	useSatShift := false
	useLumShift := false
	if n > 3 && n <= 6 {
		hueSpread = 25.0
		useSatShift = true
	} else if n > 6 {
		hueSpread = 35.0
		useSatShift = true
		useLumShift = true
	}

	colors := make([]string, n)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n-1) // 0..1
		hueOffset := (t - 0.5) * 2 * hueSpread
		hue := math.Mod(h+hueOffset+360, 360)

		si := s
		if useSatShift {
			if i%2 == 0 {
				si = math.Min(1, si+0.10)
			} else {
				si = math.Max(0.25, si-0.10)
			}
		}

		li := l
		if useLumShift {
			li += float64(i%3-1) * 0.06
			li = math.Min(0.60, math.Max(0.35, li))
		}

		colors[i] = HSLToHex(hue, si, li)
	}
	return colors
}

// GenerateThresholdVariations generates lightness variations for threshold levels of a dimension.
// Levels are ordered from lowest to highest risk; colors go from lightest to darkest.
// Matches the frontend's thresholdScale() which uses light → base → dark.
func GenerateThresholdVariations(baseHex string, levels []string) map[string]string {
	if len(levels) == 0 {
		return nil
	}

	h, s, l := HexToHSL(baseHex)
	result := make(map[string]string, len(levels))

	if len(levels) == 1 {
		result[levels[0]] = HSLToHex(h, s, l)
		return result
	}

	// Light tint → base → dark shade (matching frontend's brighten(1.8) → darken(2.2))
	lightL := math.Min(0.92, l+0.30)
	lightS := math.Min(1.0, s+0.10)
	darkL := math.Max(0.15, l-0.25)
	darkS := math.Min(1.0, s+0.15)

	for i, level := range levels {
		t := float64(i) / float64(len(levels)-1)
		var ci_l, ci_s float64
		if t <= 0.5 {
			// light → base
			u := t * 2 // 0..1
			ci_l = lightL + (l-lightL)*u
			ci_s = lightS + (s-lightS)*u
		} else {
			// base → dark
			u := (t - 0.5) * 2 // 0..1
			ci_l = l + (darkL-l)*u
			ci_s = s + (darkS-s)*u
		}
		result[level] = HSLToHex(h, ci_s, ci_l)
	}
	return result
}

// GenerateFullColorConfig generates a complete ColorConfig from a primary color
// and the questionnaire's evaluation config structure.
func GenerateFullColorConfig(primaryHex string, evalConfig *models.EvaluationConfig) *models.ColorConfig {
	if evalConfig == nil {
		return &models.ColorConfig{
			Mode:         models.ColorModeFromPrimary,
			PrimaryColor: primaryHex,
		}
	}

	if primaryHex == "" {
		primaryHex = DefaultPrimaryColor
	}

	// Generate N distinct base hues for N dimensions
	nDims := len(evalConfig.Dimensions)
	baseHues := GenerateDistinctHues(primaryHex, nDims)

	dimColors := make([]models.DimensionColorConfig, 0, nDims)
	for i, dim := range evalConfig.Dimensions {
		levels := make([]string, 0, len(dim.Thresholds))
		for _, t := range dim.Thresholds {
			levels = append(levels, t.Level)
		}

		thresholdColors := GenerateThresholdVariations(baseHues[i], levels)
		dimColors = append(dimColors, models.DimensionColorConfig{
			DimensionCode:   dim.Code,
			ThresholdColors: thresholdColors,
		})
	}

	// Generate risk profile colors: scale from medium to dark of primary
	// Matches frontend: chroma.scale([rpLight, rpDark]).colors(N)
	nProfiles := len(evalConfig.RiskProfiles)
	h, s, l := HexToHSL(primaryHex)
	rpLightS := math.Min(1.0, s+0.15)
	rpLightL := l
	rpDarkS := math.Min(1.0, s+0.30)
	rpDarkL := math.Max(0.15, l-0.35)

	rpColors := make([]models.RiskProfileColorConfig, 0, nProfiles)
	for i, rp := range evalConfig.RiskProfiles {
		t := 0.0
		if nProfiles > 1 {
			t = float64(i) / float64(nProfiles-1)
		}
		ci_s := rpLightS + (rpDarkS-rpLightS)*t
		ci_l := rpLightL + (rpDarkL-rpLightL)*t
		color := HSLToHex(h, ci_s, ci_l)
		rpColors = append(rpColors, models.RiskProfileColorConfig{
			ProfileName: rp.Name,
			Color:       color,
		})
	}

	return &models.ColorConfig{
		Mode:              models.ColorModeFromPrimary,
		PrimaryColor:      primaryHex,
		DimensionColors:   dimColors,
		RiskProfileColors: rpColors,
	}
}
