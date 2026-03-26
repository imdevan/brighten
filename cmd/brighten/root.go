package main

import (
	"fmt"
	"math"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"brighten/internal/adapters/clipboard"
	pkg "brighten/internal/package"
)

// Metadata loaded from package.toml at build time
var (
	version = pkg.Version()
	name    = pkg.Name()
	short   = pkg.Short()
)

type rootOptions struct {
	showVersion bool
}

var rootCmd = newRootCmd()

// Execute is the CLI entrypoint.
func Execute() error {
	return rootCmd.Execute()
}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{}
	cmd := &cobra.Command{
		Use:   name + " [color] [brightness]",
		Short: short,
		Long: `Adjust color brightness with format-aware brightness modes.

Supports: hex (#fff or fff), rgb(r,g,b), rgba(r,g,b,a), hsl(h,s,l), hsla(h,s,l,a)

Brightness modes (all formats):
  (no value)      auto-brighten until max channel reached
  -               auto-darken until min channel reaches 0
  50% or -50%     increase/decrease by percentage
  0.5 or 1.5      multiply by fraction
  10 or -5        add/subtract integer value

Brightness ranges by color format:
  hex: integer 0-16 (0-F) per channel
  rgb: integer 0-255 per channel
  hsl: integer 0-100 for lightness`,
		Args: cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.showVersion {
				cmd.Printf("%s\n", resolvedVersion())
				return nil
			}
			if len(args) == 0 {
				return cmd.Help()
			}
			colorArg := args[0]
			brightnessArg := ""
			if len(args) > 1 {
				brightnessArg = args[1]
			}
			return runBrighten(colorArg, brightnessArg)
		},
	}

	cmd.Flags().BoolVarP(&opts.showVersion, "version", "v", false, "print version information")
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newCompletionCmd())

	return cmd
}

func resolvedVersion() string {
	ver := version
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ver
	}
	if ver == "dev" && strings.TrimSpace(info.Main.Version) != "" && info.Main.Version != "(devel)" {
		ver = info.Main.Version
	}
	return ver
}

type colorFormat int

const (
	formatHex colorFormat = iota
	formatRGB
	formatHSL
)

type brightnessMode int

const (
	modeAuto brightnessMode = iota
	modeAutoDarken
	modePercent
	modeFraction
	modeInteger
)

var (
	reHex6     = regexp.MustCompile(`^#?([0-9a-fA-F]{6})([0-9a-fA-F]{2})?$`)
	reHex3     = regexp.MustCompile(`^#?([0-9a-fA-F]{3})$`)
	reRGB      = regexp.MustCompile(`^rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([0-9.]+))?\)$`)
	reHSL      = regexp.MustCompile(`^hsla?\((\d+),\s*(\d+)%?,\s*(\d+)%?(?:,\s*([0-9.]+))?\)$`)
	rePercent  = regexp.MustCompile(`^-?\d+%$`)
	reFraction = regexp.MustCompile(`^\d*\.\d+$`)
	reInteger  = regexp.MustCompile(`^-?\d+$`)
)

func clampInt(val, lo, hi int) int {
	if val < lo {
		return lo
	}
	if val > hi {
		return hi
	}
	return val
}

// hslToRGB converts HSL (h: 0-360, s: 0-100, l: 0-100) to RGB (0-255 each).
// Implements Python colorsys.hls_to_rgb logic (note: Python uses HLS order h,l,s).
func hslToRGB(h, s, l float64) (r, g, b int) {
	hN := h / 360.0
	sN := s / 100.0
	lN := l / 100.0

	if sN == 0 {
		v := int(lN * 255)
		return v, v, v
	}

	var m2 float64
	if lN <= 0.5 {
		m2 = lN * (1.0 + sN)
	} else {
		m2 = lN + sN - lN*sN
	}
	m1 := 2.0*lN - m2

	vFunc := func(m1, m2, hue float64) float64 {
		hue = math.Mod(hue, 1.0)
		if hue < 0 {
			hue += 1.0
		}
		if hue < 1.0/6.0 {
			return m1 + (m2-m1)*hue*6.0
		}
		if hue < 0.5 {
			return m2
		}
		if hue < 2.0/3.0 {
			return m1 + (m2-m1)*(2.0/3.0-hue)*6.0
		}
		return m1
	}

	return int(vFunc(m1, m2, hN+1.0/3.0) * 255),
		int(vFunc(m1, m2, hN)*255),
		int(vFunc(m1, m2, hN-1.0/3.0) * 255)
}

func runBrighten(colorArg, brightnessArg string) error {
	// Parse brightness mode
	var bMode brightnessMode
	var bValue string

	switch {
	case brightnessArg == "":
		bMode = modeAuto
	case brightnessArg == "-":
		bMode = modeAutoDarken
	case rePercent.MatchString(brightnessArg):
		bMode = modePercent
		bValue = strings.TrimSuffix(brightnessArg, "%")
	case reFraction.MatchString(brightnessArg):
		bMode = modeFraction
		bValue = brightnessArg
	case reInteger.MatchString(brightnessArg):
		bMode = modeInteger
		bValue = brightnessArg
	default:
		return fmt.Errorf("Error: Invalid brightness format")
	}

	// Parse color
	var r, g, b, h, s, l int
	var alpha string
	var format colorFormat

	switch {
	case reHex6.MatchString(colorArg):
		m := reHex6.FindStringSubmatch(colorArg)
		hex := m[1]
		alpha = m[2]
		rv, _ := strconv.ParseInt(hex[0:2], 16, 64)
		gv, _ := strconv.ParseInt(hex[2:4], 16, 64)
		bv, _ := strconv.ParseInt(hex[4:6], 16, 64)
		r, g, b = int(rv), int(gv), int(bv)
		format = formatHex

	case reHex3.MatchString(colorArg):
		m := reHex3.FindStringSubmatch(colorArg)
		hex := m[1]
		rv, _ := strconv.ParseInt(string(hex[0])+string(hex[0]), 16, 64)
		gv, _ := strconv.ParseInt(string(hex[1])+string(hex[1]), 16, 64)
		bv, _ := strconv.ParseInt(string(hex[2])+string(hex[2]), 16, 64)
		r, g, b = int(rv), int(gv), int(bv)
		format = formatHex

	case reRGB.MatchString(colorArg):
		m := reRGB.FindStringSubmatch(colorArg)
		r, _ = strconv.Atoi(m[1])
		g, _ = strconv.Atoi(m[2])
		b, _ = strconv.Atoi(m[3])
		alpha = m[4]
		format = formatRGB

	case reHSL.MatchString(colorArg):
		m := reHSL.FindStringSubmatch(colorArg)
		h, _ = strconv.Atoi(m[1])
		s, _ = strconv.Atoi(m[2])
		l, _ = strconv.Atoi(m[3])
		alpha = m[4]
		format = formatHSL

	default:
		return fmt.Errorf("Error: Unsupported color format")
	}

	// Apply brightness adjustment
	var newR, newG, newB, newH, newS, newL int

	if format == formatHSL {
		newH = h
		newS = s

		switch bMode {
		case modeAuto:
			newL = 100
		case modeAutoDarken:
			newL = 0
		case modePercent:
			pct, _ := strconv.ParseFloat(bValue, 64)
			newL = clampInt(int(float64(l)*(1+pct/100)), 0, 100)
		case modeFraction:
			frac, _ := strconv.ParseFloat(bValue, 64)
			newL = clampInt(int(float64(l)*frac), 0, 100)
		case modeInteger:
			add, _ := strconv.Atoi(bValue)
			newL = clampInt(l+add, 0, 100)
		}

		// Convert HSL to RGB for ANSI color display
		r, g, b = hslToRGB(float64(h), float64(s), float64(l))
		newR, newG, newB = hslToRGB(float64(newH), float64(newS), float64(newL))

	} else {
		// hex and rgb share the same brightness logic (0-255 channels)
		switch bMode {
		case modeAuto:
			// Scale up so the max channel reaches 255
			maxCh := r
			maxIdx := 0
			if g > maxCh {
				maxCh = g
				maxIdx = 1
			}
			if b > maxCh {
				maxCh = b
				maxIdx = 2
			}
			if maxCh == 0 {
				newR, newG, newB = 0, 0, 0
			} else {
				mult := 255.0 / float64(maxCh)
				newR = clampInt(int(float64(r)*mult), 0, 255)
				newG = clampInt(int(float64(g)*mult), 0, 255)
				newB = clampInt(int(float64(b)*mult), 0, 255)
				// Fix floating point precision on the max channel
				switch maxIdx {
				case 0:
					newR = 255
				case 1:
					newG = 255
				case 2:
					newB = 255
				}
			}

		case modeAutoDarken:
			// Scale down so the min channel reaches 0
			// Formula: new = 255 - ((255 - old) * 255 / (255 - min))
			minCh := r
			minIdx := 0
			if g < minCh {
				minCh = g
				minIdx = 1
			}
			if b < minCh {
				minCh = b
				minIdx = 2
			}
			if minCh == 255 {
				newR, newG, newB = 255, 255, 255
			} else {
				newR = 255 - (255-r)*255/(255-minCh)
				newG = 255 - (255-g)*255/(255-minCh)
				newB = 255 - (255-b)*255/(255-minCh)
				// Fix integer division precision on the min channel
				switch minIdx {
				case 0:
					newR = 0
				case 1:
					newG = 0
				case 2:
					newB = 0
				}
			}

		case modePercent:
			pct, _ := strconv.ParseFloat(bValue, 64)
			newR = clampInt(int(float64(r)*(1+pct/100)), 0, 255)
			newG = clampInt(int(float64(g)*(1+pct/100)), 0, 255)
			newB = clampInt(int(float64(b)*(1+pct/100)), 0, 255)

		case modeFraction:
			frac, _ := strconv.ParseFloat(bValue, 64)
			newR = clampInt(int(float64(r)*frac), 0, 255)
			newG = clampInt(int(float64(g)*frac), 0, 255)
			newB = clampInt(int(float64(b)*frac), 0, 255)

		case modeInteger:
			add, _ := strconv.Atoi(bValue)
			newR = clampInt(r+add, 0, 255)
			newG = clampInt(g+add, 0, 255)
			newB = clampInt(b+add, 0, 255)
		}
	}

	// Format output strings
	var oldColor, newColor string
	switch format {
	case formatHex:
		oldColor = fmt.Sprintf("#%02x%02x%02x", r, g, b)
		newColor = fmt.Sprintf("#%02x%02x%02x", newR, newG, newB)
		if alpha != "" {
			oldColor += alpha
		}
	case formatHSL:
		if alpha != "" {
			oldColor = fmt.Sprintf("hsla(%d, %d%%, %d%%, %s)", h, s, l, alpha)
			newColor = fmt.Sprintf("hsla(%d, %d%%, %d%%, %s)", newH, newS, newL, alpha)
		} else {
			oldColor = fmt.Sprintf("hsl(%d, %d%%, %d%%)", h, s, l)
			newColor = fmt.Sprintf("hsl(%d, %d%%, %d%%)", newH, newS, newL)
		}
	case formatRGB:
		if alpha != "" {
			oldColor = fmt.Sprintf("rgba(%d, %d, %d, %s)", r, g, b, alpha)
			newColor = fmt.Sprintf("rgba(%d, %d, %d, %s)", newR, newG, newB, alpha)
		} else {
			oldColor = fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
			newColor = fmt.Sprintf("rgb(%d, %d, %d)", newR, newG, newB)
		}
	}

	// Copy new color to clipboard
	cb := clipboard.Adapter{}
	_ = cb.WriteText(newColor)

	// Print results with ANSI truecolor
	fmt.Printf("\033[32m✓\033[0m Old: \033[38;2;%d;%d;%dm%s\033[0m\n", r, g, b, oldColor)
	fmt.Printf("\033[32m✓\033[0m New: \033[38;2;%d;%d;%dm%s\033[0m (copied to clipboard)\n", newR, newG, newB, newColor)

	return nil
}
