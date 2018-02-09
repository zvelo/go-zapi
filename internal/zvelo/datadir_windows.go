package zvelo

import (
	"os"
	"path/filepath"
)

// C:\Users\<username>\AppData\Local
func DataDir(name string) string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), name)
}
