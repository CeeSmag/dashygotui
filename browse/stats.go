package browse

import (
	"errors"
	"io/fs"
	"path/filepath"
	"time"
)

type dirStats struct {
	files int64
	bytes int64
	ok    bool
}

func (n *Navigator) statsFor(path string, mod time.Time) dirStats {
	key := path + "\x00" + mod.UTC().Format(time.RFC3339Nano)
	if st, ok := n.cache[key]; ok {
		return st
	}
	st := walkStats(path)
	n.cache[key] = st
	return st
}

// walkFileCap stops a recursive walk after this many regular files so listing
// a huge tree cannot freeze the UI.
const walkFileCap = 10000

func walkStats(root string) dirStats {
	var files, bytes int64
	ok := true
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			ok = false
			if d != nil && d.IsDir() && path != root {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			ok = false
			return nil //nolint:nilerr // skip this entry; ok records the partial failure
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		files++
		bytes += info.Size()
		if files >= walkFileCap {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		ok = false
	}
	if !ok && files == 0 && bytes == 0 {
		return dirStats{files: -1, bytes: -1, ok: false}
	}
	return dirStats{files: files, bytes: bytes, ok: ok}
}
