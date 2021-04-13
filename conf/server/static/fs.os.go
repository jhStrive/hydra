package static

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

//osfs 本地文件系统
type osfs struct {
	dir string
	fs  http.FileSystem
}

type localFs string

func containsAny(s, chars string) bool {
	for i := 0; i < len(s); i++ {
		for j := 0; j < len(chars); j++ {
			if s[i] == chars[j] {
				return true
			}
		}
	}
	return false
}
func (dir localFs) Open(name string) (fs.File, error) {
	f, err := os.Open(filepath.Join(string(dir), name))
	if err != nil {
		return nil, err
	}
	return f, nil
}

func newOSFS(dir string) *osfs {
	return &osfs{
		dir: dir,
		fs:  http.FS(localFs(dir)),
	}
}

func (o *osfs) ReadFile(name string) (http.FileSystem, string, error) {
	return o.fs, name, nil
}

func (o *osfs) GetRoot() string {
	return o.dir
}

func (o *osfs) Has(name string) bool {
	info, err := os.Stat(filepath.Join(o.dir, name))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (o *osfs) GetDirEntrys(path string) (dirs []fs.DirEntry, err error) {
	return os.ReadDir(filepath.Join(o.dir, path))
}
