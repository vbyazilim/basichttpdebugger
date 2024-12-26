package writerutils

import (
	"io"
	"os"
)

// GetFilePathName returns given writers physical file path name.
func GetFilePathName(w io.Writer) string {
	if file, ok := w.(*os.File); ok {
		return file.Name()
	}

	return ""
}
