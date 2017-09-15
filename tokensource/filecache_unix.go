// +build !windows

// NOTE: technically darwin doesn't support XDG, but if neovim uses it, that's
// good enough

package tokensource

import (
	"os"
	"path/filepath"
)

// https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html
var dataDir string

func init() {
	if xdh := os.Getenv("XDG_DATA_HOME"); xdh != "" {
		dataDir = xdh
		return
	}

	dataDir = filepath.Join(os.Getenv("HOME"), ".local", "share")
}
