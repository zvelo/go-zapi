package zvelo

import "os"

// C:\Users\<username>\AppData\Local
var DataDir = os.Getenv("LOCALAPPDATA")
