package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout during fn and returns what was printed.
func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// --- clampInt ---

func TestClampInt(t *testing.T) {
	tests := []struct {
		val, lo, hi, want int
	}{
		{50, 0, 100, 50},
		{-10, 0, 255, 0},
		{300, 0, 255, 255},
		{0, 0, 100, 0},
		{100, 0, 100, 100},
	}
	for _, tt := range tests {
		if got := clampInt(tt.val, tt.lo, tt.hi); got != tt.want {
			t.Errorf("clampInt(%d, %d, %d) = %d, want %d", tt.val, tt.lo, tt.hi, got, tt.want)
		}
	}
}

// --- hslToRGB ---

func TestHslToRGB(t *testing.T) {
	tests := []struct {
		h, s, l float64
		r, g, b int
		desc    string
	}{
		{0, 100, 50, 255, 0, 0, "red"},
		{120, 100, 50, 0, 255, 0, "green"},
		{240, 100, 50, 0, 0, 255, "blue"},
		{0, 0, 100, 255, 255, 255, "white"},
		{0, 0, 0, 0, 0, 0, "black"},
		{0, 0, 50, 127, 127, 127, "gray"},
	}
	for _, tt := range tests {
		r, g, b := hslToRGB(tt.h, tt.s, tt.l)
		if r != tt.r || g != tt.g || b != tt.b {
			t.Errorf("hslToRGB(%v, %v, %v) [%s] = (%d,%d,%d), want (%d,%d,%d)",
				tt.h, tt.s, tt.l, tt.desc, r, g, b, tt.r, tt.g, tt.b)
		}
	}
}

// --- error cases ---

func TestRunBrighten_InvalidColor(t *testing.T) {
	err := runBrighten("notacolor", "")
	if err == nil || !strings.Contains(err.Error(), "Unsupported color format") {
		t.Errorf("expected unsupported color error, got: %v", err)
	}
}

func TestRunBrighten_InvalidBrightness(t *testing.T) {
	err := runBrighten("#ff0000", "abc")
	if err == nil || !strings.Contains(err.Error(), "Invalid brightness format") {
		t.Errorf("expected invalid brightness error, got: %v", err)
	}
}

// --- hex format ---

func TestRunBrighten_HexAutoMaxBrighten(t *testing.T) {
	// #4080c0: max channel b=192, scale -> r=85=0x55, g=170=0xaa, b=255
	out := captureStdout(func() { runBrighten("#4080c0", "") })
	if !strings.Contains(out, "#55aaff") {
		t.Errorf("unexpected auto-brighten output: %q", out)
	}
	if !strings.Contains(out, "copied to clipboard") {
		t.Errorf("missing clipboard message: %q", out)
	}
}

func TestRunBrighten_HexAutoDarken(t *testing.T) {
	// #4080c0: min channel r=64
	// new_r=0, new_g=255-(127*255/191)=86=0x56, new_b=255-(63*255/191)=171=0xab
	out := captureStdout(func() { runBrighten("#4080c0", "-") })
	if !strings.Contains(out, "#0056ab") {
		t.Errorf("unexpected auto-darken output: %q", out)
	}
}

func TestRunBrighten_HexPercent(t *testing.T) {
	// #808080 + 50% -> 128*1.5=192=0xc0
	out := captureStdout(func() { runBrighten("#808080", "50%") })
	if !strings.Contains(out, "#c0c0c0") {
		t.Errorf("unexpected percent output: %q", out)
	}
}

func TestRunBrighten_HexNegativePercent(t *testing.T) {
	// #808080 - 50% -> 128*0.5=64=0x40
	out := captureStdout(func() { runBrighten("#808080", "-50%") })
	if !strings.Contains(out, "#404040") {
		t.Errorf("unexpected negative percent output: %q", out)
	}
}

func TestRunBrighten_HexFraction(t *testing.T) {
	// #808080 * 0.5 -> 64=0x40
	out := captureStdout(func() { runBrighten("#808080", "0.5") })
	if !strings.Contains(out, "#404040") {
		t.Errorf("unexpected fraction output: %q", out)
	}
}

func TestRunBrighten_HexInteger(t *testing.T) {
	// #080808 + 10 -> 18=0x12
	out := captureStdout(func() { runBrighten("#080808", "10") })
	if !strings.Contains(out, "#121212") {
		t.Errorf("unexpected integer output: %q", out)
	}
}

func TestRunBrighten_HexIntegerClamp(t *testing.T) {
	// #f0f0f0 + 50 -> clamped to 0xff
	out := captureStdout(func() { runBrighten("#f0f0f0", "50") })
	if !strings.Contains(out, "#ffffff") {
		t.Errorf("expected clamp to #ffffff: %q", out)
	}
}

func TestRunBrighten_HexShortForm(t *testing.T) {
	// #fff -> expands to #ffffff, auto-brighten stays #ffffff
	out := captureStdout(func() { runBrighten("#fff", "") })
	if !strings.Contains(out, "#ffffff") {
		t.Errorf("unexpected short hex output: %q", out)
	}
}

func TestRunBrighten_HexNoHash(t *testing.T) {
	out := captureStdout(func() { runBrighten("808080", "0") })
	if !strings.Contains(out, "#808080") {
		t.Errorf("unexpected no-hash hex output: %q", out)
	}
}

// --- rgb format ---

func TestRunBrighten_RGBAutoMaxBrighten(t *testing.T) {
	// rgb(64, 128, 192): max b=192, scale -> (85, 170, 255)
	out := captureStdout(func() { runBrighten("rgb(64, 128, 192)", "") })
	if !strings.Contains(out, "rgb(85, 170, 255)") {
		t.Errorf("unexpected rgb auto-brighten output: %q", out)
	}
}

func TestRunBrighten_RGBInteger(t *testing.T) {
	out := captureStdout(func() { runBrighten("rgb(100, 100, 100)", "55") })
	if !strings.Contains(out, "rgb(155, 155, 155)") {
		t.Errorf("unexpected rgb integer output: %q", out)
	}
}

func TestRunBrighten_RGBAAlphaPreserved(t *testing.T) {
	out := captureStdout(func() { runBrighten("rgba(64, 128, 192, 0.5)", "0") })
	if !strings.Contains(out, "0.5") {
		t.Errorf("alpha not preserved in rgba output: %q", out)
	}
}

// --- hsl format ---

func TestRunBrighten_HSLAutoMaxBrighten(t *testing.T) {
	out := captureStdout(func() { runBrighten("hsl(210, 50%, 50%)", "") })
	if !strings.Contains(out, "hsl(210, 50%, 100%)") {
		t.Errorf("unexpected hsl auto-brighten output: %q", out)
	}
}

func TestRunBrighten_HSLAutoDarken(t *testing.T) {
	out := captureStdout(func() { runBrighten("hsl(210, 50%, 50%)", "-") })
	if !strings.Contains(out, "hsl(210, 50%, 0%)") {
		t.Errorf("unexpected hsl auto-darken output: %q", out)
	}
}

func TestRunBrighten_HSLInteger(t *testing.T) {
	out := captureStdout(func() { runBrighten("hsl(210, 50%, 40%)", "20") })
	if !strings.Contains(out, "hsl(210, 50%, 60%)") {
		t.Errorf("unexpected hsl integer output: %q", out)
	}
}

func TestRunBrighten_HSLIntegerClamp(t *testing.T) {
	out := captureStdout(func() { runBrighten("hsl(210, 50%, 90%)", "20") })
	if !strings.Contains(out, "hsl(210, 50%, 100%)") {
		t.Errorf("expected hsl clamp to 100%%: %q", out)
	}
}

func TestRunBrighten_HSLPercent(t *testing.T) {
	// l=50 + 100% -> 50*(1+1.0)=100
	out := captureStdout(func() { runBrighten("hsl(210, 50%, 50%)", "100%") })
	if !strings.Contains(out, "hsl(210, 50%, 100%)") {
		t.Errorf("unexpected hsl percent output: %q", out)
	}
}

func TestRunBrighten_HSLFraction(t *testing.T) {
	// l=50 * 0.5 = 25
	out := captureStdout(func() { runBrighten("hsl(210, 50%, 50%)", "0.5") })
	if !strings.Contains(out, "hsl(210, 50%, 25%)") {
		t.Errorf("unexpected hsl fraction output: %q", out)
	}
}

func TestRunBrighten_HSLAAlphaPreserved(t *testing.T) {
	out := captureStdout(func() { runBrighten("hsla(210, 50%, 50%, 0.8)", "10") })
	if !strings.Contains(out, "0.8") {
		t.Errorf("alpha not preserved in hsla output: %q", out)
	}
}
