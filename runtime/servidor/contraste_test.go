package servidor

import (
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func luminance(hex string) float64 {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	ch := func(i int) float64 {
		v, _ := strconv.ParseUint(hex[i:i+2], 16, 8)
		c := float64(v) / 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(0) + 0.7152*ch(2) + 0.0722*ch(4)
}

func contrast(a, b string) float64 {
	la, lb := luminance(a), luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// The colors of the generated pages meet WCAG 2.2 in both modes: text on its
// background 4.5:1 (1.4.3), field borders 3:1 against the page (1.4.11).
func TestContrasteDasPaginas(t *testing.T) {
	raw, err := os.ReadFile("paginas.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	src = src[strings.Index(src, "var layoutTpl"):]
	vars := func(block string) map[string]string {
		out := map[string]string{}
		for _, m := range regexp.MustCompile(`--([a-z-]+):(#[0-9a-f]{6}|#[0-9a-f]{3})\b`).FindAllStringSubmatch(block, -1) {
			out[m[1]] = m[2]
		}
		return out
	}
	light := vars(src[strings.Index(src, ":root{"):strings.Index(src, "@media (prefers-color-scheme:dark){:root")])
	dark := map[string]string{}
	for k, v := range light {
		dark[k] = v
	}
	d := src[strings.Index(src, "@media (prefers-color-scheme:dark){:root"):]
	for k, v := range vars(d[:strings.Index(d, "}}")]) {
		dark[k] = v
	}
	for mode, c := range map[string]map[string]string{"claro": light, "escuro": dark} {
		for _, pair := range [][3]string{
			{"fg", "bg", "4.5"}, {"muted", "bg", "4.5"}, {"on-accent", "accent", "4.5"}, {"on-bad", "bad", "4.5"}, {"accent", "bg", "4.5"}, {"field", "bg", "3"},
		} {
			want, _ := strconv.ParseFloat(pair[2], 64)
			if got := contrast(c[pair[0]], c[pair[1]]); got < want {
				t.Errorf("modo %s: %s sobre %s = %.2f:1, mínimo %s:1", mode, pair[0], pair[1], got, pair[2])
			}
		}
	}
}

// Text on the theme's primary color (the older interface) reaches the best
// contrast available for every named color.
func TestTextoSobreCorDoTema(t *testing.T) {
	for _, bg := range []string{"#3b82f6", "#eab308", "#6366f1", "#22c55e", "#ef4444", "#f97316", "#ec4899", "#14b8a6", "#8b5cf6", "#6b7280"} {
		fg := onColor(bg)
		if got := contrast(fg, bg); got < 4.5 {
			t.Errorf("%s sobre %s = %.2f:1", fg, bg, got)
		}
	}
}
