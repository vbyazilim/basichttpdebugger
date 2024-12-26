package writerutils_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/vbyazilim/basichttpdebugger/internal/writerutils"
)

func TestGetFilePathName(t *testing.T) {
	tests := []struct {
		name     string
		writer   io.Writer
		wantPath string
	}{
		{
			name:     "Standard Output",
			writer:   os.Stdout,
			wantPath: os.Stdout.Name(),
		},
		{
			name:     "Standard Error",
			writer:   os.Stderr,
			wantPath: os.Stderr.Name(),
		},
		{
			name: "Regular File",
			writer: func() io.Writer {
				file, err := os.Create(filepath.Clean("/tmp/testfile"))
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				t.Cleanup(func() { os.Remove("/tmp/testfile") })
				return file
			}(),
			wantPath: "/tmp/testfile",
		},
		{
			name:     "Non-File Writer",
			writer:   &bytes.Buffer{},
			wantPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writerutils.GetFilePathName(tt.writer)
			if got != tt.wantPath {
				t.Errorf("GetFilePathName() = %q, want %q", got, tt.wantPath)
			}
		})
	}
}
