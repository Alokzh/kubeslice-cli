package util

import (
	"io"
	"os"
	"time"
)

var (
	CommandExecutor Executor       = &defaultExecutor{}
	FileSystem      FileOperations = &defaultFileSystem{}
	Output          OutputWriter   = &defaultOutput{}
	SystemClock     Clock          = &defaultClock{}
)

// Executor defines the interface for executing external commands.
type Executor interface {
	Execute(cli string, args ...string) error
	ExecuteWithOutput(cli string, stdout, stderr io.Writer, args ...string) error
}

// FileOperations defines the interface for filesystem operations.
type FileOperations interface {
	Stat(name string) (os.FileInfo, error)
	Create(name string) (*os.File, error)
	WriteFile(filename string, data []byte, perm os.FileMode) error
	ReadFile(filename string) ([]byte, error)
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
}

// OutputWriter defines the interface for logging operations.
type OutputWriter interface {
	Infof(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// Clock defines the interface for time operations.
type Clock interface {
	Sleep(d time.Duration)
	Now() time.Time
}
