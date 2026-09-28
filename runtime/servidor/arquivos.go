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
