package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"
)

var stdoutMutex sync.Mutex

func captureOutput(f func()) string {
	stdoutMutex.Lock()
	defer stdoutMutex.Unlock()

	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		w.Close()
		os.Stdout = originalStdout
	}()

	f()
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "Printf with no arguments",
			format:   "Hello World",
			args:     []interface{}{},
			expected: "Hello World\n",
		},
		{
			name:     "Printf with single argument",
			format:   "Hello %s",
			args:     []interface{}{"World"},
			expected: "Hello World\n",
		},
		{
			name:     "Printf with multiple arguments",
			format:   "Error %d: %s",
			args:     []interface{}{404, "not found"},
			expected: "Error 404: not found\n",
		},
		{
			name:     "Printf with empty string",
			format:   "",
			args:     []interface{}{},
			expected: "\n",
		},
		{
			name:     "Printf with nil args slice",
			format:   "Hello %s",
			args:     nil,
			expected: "Hello %s\n",
		},
		{
			name:     "Printf with unicode constants",
			format:   "Status: %s Success: %s",
			args:     []interface{}{Cross, Tick},
			expected: fmt.Sprintf("Status: %s Success: %s\n", Cross, Tick),
		},
		{
			name:     "Printf with mixed types",
			format:   "int: %d, string: %s, bool: %t",
			args:     []interface{}{42, "hello", true},
			expected: "int: 42, string: hello, bool: true\n",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output := captureOutput(func() {
				Printf(tc.format, tc.args...)
			})
			if output != tc.expected {
				t.Errorf("Printf() output mismatch\nwant: %q\ngot:  %q", tc.expected, output)
			}
		})
	}
}

func TestFatalf(t *testing.T) {

	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "Fatalf with no arguments",
			format:   "Error occurred",
			args:     []interface{}{},
			expected: "Error occurred\n\n",
		},
		{
			name:     "Fatalf with single argument",
			format:   "Error: %s",
			args:     []interface{}{"file not found"},
			expected: "Error: file not found\n",
		},
		{
			name:     "Fatalf with multiple arguments",
			format:   "Error %d: %s",
			args:     []interface{}{404, "not found"},
			expected: "Error 404: not found\n",
		},
		{
			name:     "Fatalf with empty string",
			format:   "",
			args:     []interface{}{},
			expected: "\n\n",
		},
		{
			name:     "Fatalf with nil args slice",
			format:   "Fatal error %s",
			args:     nil,
			expected: "Fatal error %s\n\n",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if os.Getenv("BE_FATALF_SUBPROCESS") == "1" {
				Fatalf(tc.format, tc.args...)
				return
			}

			cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$")
			cmd.Env = append(os.Environ(), "BE_FATALF_SUBPROCESS=1")

			output, err := cmd.CombinedOutput()
			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("Expected command to fail with an *exec.ExitError, but it didn't. Error: %v", err)
			}

			if exitErr.ExitCode() != 1 {
				t.Errorf("expected exit code 1, but got %d", exitErr.ExitCode())
			}

			if string(output) != tc.expected {
				t.Errorf("Fatalf() output mismatch\nwant: %q\ngot:  %q", tc.expected, string(output))
			}
		})
	}
}
