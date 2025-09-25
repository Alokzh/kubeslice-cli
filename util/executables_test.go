package util

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
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
		os.Stdout = originalStdout
	}()

	f()
	w.Close()

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	switch os.Getenv("MOCK_COMMAND_BEHAVIOR") {
	case "success":
		fmt.Fprint(os.Stdout, "SUCCESS")
		os.Exit(0)
	case "fail":
		fmt.Fprint(os.Stderr, "FAILURE")
		os.Exit(1)
	case "success_with_args":
		fmt.Fprintf(os.Stdout, "SUCCESS: %s", strings.Join(os.Args[1:], " "))
		os.Exit(0)
	case "stderr_only":
		fmt.Fprint(os.Stderr, "WARNING: A non-fatal warning")
		os.Exit(0)
	default:
		fmt.Fprint(os.Stderr, "Unknown mock command behavior")
		os.Exit(127)
	}
}

func setupMock(t *testing.T, behavior string) {
	t.Helper()

	originalPaths := ExecutablePaths
	ExecutablePaths = map[string]string{
		"mock-cli": os.Args[0],
	}

	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv("MOCK_COMMAND_BEHAVIOR", behavior)
	t.Cleanup(func() {
		ExecutablePaths = originalPaths
	})
}
func TestRunCommand(t *testing.T) {
	tests := []struct {
		name                     string
		args                     []string
		mockBehavior             string
		expectError              bool
		expectOutputToContain    []string
		expectOutputToNotContain []string
	}{
		{
			name:                     "Success case with expected logging",
			args:                     []string{"test-arg"},
			mockBehavior:             "success",
			expectError:              false,
			expectOutputToContain:    []string{"Running command:"},
			expectOutputToNotContain: []string{"Failed to run command"},
		},
		{
			name:                  "Failure case with error logging",
			args:                  []string{"test-arg"},
			mockBehavior:          "fail",
			expectError:           true,
			expectOutputToContain: []string{"Failed to run command", "FAILURE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMock(t, tt.mockBehavior)

			output := captureOutput(func() {
				err := RunCommand("mock-cli", tt.args...)
				if (err != nil) != tt.expectError {
					t.Errorf("RunCommand() error = %v, expectError %v", err, tt.expectError)
				}
			})

			for _, expected := range tt.expectOutputToContain {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got: %q", expected, output)
				}
			}
			for _, notExpected := range tt.expectOutputToNotContain {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to not contain %q, but got: %q", notExpected, output)
				}
			}
		})
	}
}

func TestRunCommand_EdgeCases(t *testing.T) {
	t.Run("nil ExecutablePaths", func(t *testing.T) {
		originalPaths := ExecutablePaths
		ExecutablePaths = nil
		t.Cleanup(func() {
			ExecutablePaths = originalPaths
		})

		output := captureOutput(func() {
			err := RunCommand("any-cli")
			if err == nil {
				t.Error("Expected error when ExecutablePaths is nil, but got nil")
			}
		})

		if !strings.Contains(output, "Failed to run command") {
			t.Error("Expected failure log for nil ExecutablePaths")
		}
	})
}

func TestRunCommandWithoutPrint(t *testing.T) {
	tests := []struct {
		name         string
		mockBehavior string
		expectError  bool
	}{
		{"Success case", "success", false},
		{"Failure case", "fail", true},
		{"Success with stderr output", "stderr_only", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMock(t, tt.mockBehavior)

			output := captureOutput(func() {
				err := RunCommandWithoutPrint("mock-cli")
				if (err != nil) != tt.expectError {
					t.Errorf("RunCommandWithoutPrint() error = %v, expectError %v", err, tt.expectError)
				}
			})

			if output != "" {
				t.Errorf("Expected no output from RunCommandWithoutPrint, but got: %s", output)
			}
		})
	}
}

func TestRunCommandOnStdIO(t *testing.T) {
	setupMock(t, "success")

	output := captureOutput(func() {
		err := RunCommandOnStdIO("mock-cli")
		if err != nil {
			t.Fatalf("Expected command to succeed, but got error: %v", err)
		}
	})

	if !strings.Contains(output, "Running command:") {
		t.Error("Expected 'Running command:' log, but it was not found")
	}
	if !strings.Contains(output, "SUCCESS") {
		t.Errorf("Expected command output 'SUCCESS', but got: %s", output)
	}
}

func TestRunCommandCustomIO(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		mockBehavior  string
		suppressPrint bool
		expectError   bool
		validateFunc  func(t *testing.T, stdout, stderr, wrapperOutput string)
	}{
		{
			name:          "Success with multiple args and print enabled",
			args:          []string{"arg1", "arg2"},
			mockBehavior:  "success_with_args",
			suppressPrint: false,
			expectError:   false,
			validateFunc: func(t *testing.T, stdout, stderr, wrapperOutput string) {
				if !strings.Contains(wrapperOutput, "Running command:") {
					t.Error("Expected wrapper output to contain 'Running command:' log")
				}
				if stdout != "SUCCESS: arg1 arg2" {
					t.Errorf("Expected stdout to be 'SUCCESS: arg1 arg2', got %q", stdout)
				}
			},
		},
		{
			name:          "Success with print suppressed",
			args:          []string{},
			mockBehavior:  "success",
			suppressPrint: true,
			expectError:   false,
			validateFunc: func(t *testing.T, stdout, stderr, wrapperOutput string) {
				if wrapperOutput != "" {
					t.Errorf("Expected no wrapper output, but got: %q", wrapperOutput)
				}
				if stdout != "SUCCESS" {
					t.Errorf("Expected stdout to be 'SUCCESS', got %q", stdout)
				}
			},
		},
		{
			name:          "Failure with stderr capture",
			args:          []string{},
			mockBehavior:  "fail",
			suppressPrint: false,
			expectError:   true,
			validateFunc: func(t *testing.T, stdout, stderr, wrapperOutput string) {
				if stderr != "FAILURE" {
					t.Errorf("Expected stderr to be 'FAILURE', got %q", stderr)
				}
			},
		},
		{
			name:          "Success with stderr output capture",
			args:          []string{},
			mockBehavior:  "stderr_only",
			suppressPrint: false,
			expectError:   false,
			validateFunc: func(t *testing.T, stdout, stderr, wrapperOutput string) {
				if !strings.Contains(stderr, "WARNING: A non-fatal warning") {
					t.Errorf("Expected stderr buffer to contain warning, got %q", stderr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupMock(t, tt.mockBehavior)
			var outB, errB bytes.Buffer

			wrapperOutput := captureOutput(func() {
				err := RunCommandCustomIO("mock-cli", &outB, &errB, tt.suppressPrint, tt.args...)
				if (err != nil) != tt.expectError {
					t.Errorf("RunCommandCustomIO() error = %v, expectError %v", err, tt.expectError)
				}
			})

			if tt.validateFunc != nil {
				tt.validateFunc(t, outB.String(), errB.String(), wrapperOutput)
			}
		})
	}
}

func TestRunCommandCustomIO_EdgeCases(t *testing.T) {
	t.Run("nil stdout and stderr writers", func(t *testing.T) {
		setupMock(t, "success")
		err := RunCommandCustomIO("mock-cli", nil, nil, true, "test")
		if err != nil {
			t.Errorf("Expected no error with nil writers, got: %v", err)
		}
	})
}

func TestExecutableVerifyCommands(t *testing.T) {
	t.Parallel()
	expected := map[string][]string{
		"kind":    {"version"},
		"kubectl": {"version", "--client=true"},
		"docker":  {"ps", "-a"},
		"helm":    {"version"},
	}

	if !reflect.DeepEqual(expected, ExecutableVerifyCommands) {
		t.Errorf("ExecutableVerifyCommands map does not match expected values.\nWant: %v\nGot:  %v", expected, ExecutableVerifyCommands)
	}
}
