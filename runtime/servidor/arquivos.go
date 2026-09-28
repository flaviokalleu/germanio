package servidor

import (
	"io/fs"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// fileServer serves the files of a folder without directory listings and
// without hidden files (names starting with a dot): a listing of /uploads/
// would let anyone enumerate other people's files.
func fileServer(dir string) http.Handler {
	return http.FileServer(filesOnly{http.Dir(dir)})
}

type filesOnly struct{ fs http.FileSystem }

func (f filesOnly) Open(name string) (http.File, error) {
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") {
			return nil, fs.ErrNotExist
		}
	}
	file, err := f.fs.Open(name)
	if err != nil {
		return nil, err
	}
	if st, err := file.Stat(); err != nil || st.IsDir() {
		file.Close()
		return nil, fs.ErrNotExist
	}
	return file, nil
}

// uploadsServer serves what people sent. Those files are never trusted as
// part of the site: the sandbox policy keeps a script inside an SVG (or any
// file a browser might render) from running with the site's cookies, nosniff
// stops the browser from guessing a dangerous type, and SVG is a download.
func uploadsServer(dir string) http.Handler {
	files := fileServer(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self'; style-src 'unsafe-inline'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if strings.HasSuffix(strings.ToLower(r.URL.Path), ".svg") {
			w.Header().Set("Content-Disposition", "attachment")
		}
		files.ServeHTTP(w, r)
	})
}

// onColor picks the text color for a background color: white or near-black,
// whichever has the higher contrast (WCAG 2.2 formula). White text on a
// theme color used to fall below 4.5:1 (blue 3.68, yellow 1.92).
func onColor(bg string) string {
	const dark = "#000000"
	if contrastRatio("#ffffff", bg) >= contrastRatio(dark, bg) {
		return "#ffffff"
	}
	return dark
}

func contrastRatio(a, b string) float64 {
	la, lb := relLuminance(a), relLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relLuminance(hex string) float64 {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return 0
	}
	ch := func(i int) float64 {
		v, err := strconv.ParseUint(hex[i:i+2], 16, 8)
		if err != nil {
			return 0
		}
		c := float64(v) / 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(0) + 0.7152*ch(2) + 0.0722*ch(4)
}
