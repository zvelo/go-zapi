package tokensource

import "os"

// C:\Users\<username>\AppData\Local
var dataDir = os.Getenv("LOCALAPPDATA")
