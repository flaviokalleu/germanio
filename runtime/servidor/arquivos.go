package servidor

import (
	"io/fs"
	"net/http"
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
