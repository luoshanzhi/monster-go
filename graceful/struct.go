package graceful

import (
	"os"
)

type File struct {
	File *os.File
	Addr string
}
