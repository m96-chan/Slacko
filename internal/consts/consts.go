package consts

import (
	"os"
	"path/filepath"
)

const Name = "slacko"

var CacheDir string

func init() {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	CacheDir = filepath.Join(dir, Name)
	os.MkdirAll(CacheDir, 0o700)
}
