// +build !windows

// NOTE: technically darwin doesn't support XDG, but if neovim uses it, that's
// good enough

package zvelo

import (
	"os"
	"path/filepath"
)

// https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html
func DataDir(name string) string {
	if dir := os.Getenv("SNAP_USER_COMMON"); dir != "" {
		// the dir is already specific to the app, so don't append `name`
		return dir
	}

	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, name)
	}

	return filepath.Join(os.Getenv("HOME"), ".local", "share", name)
}
