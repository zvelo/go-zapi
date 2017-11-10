// +build !windows

// NOTE: technically darwin doesn't support XDG, but if neovim uses it, that's
// good enough

package zvelo

import (
	"os"
	"path/filepath"
)

// https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html
var DataDir string

func init() {
	if xdh := os.Getenv("XDG_DATA_HOME"); xdh != "" {
		DataDir = xdh
		return
	}

	DataDir = filepath.Join(os.Getenv("HOME"), ".local", "share")
}
