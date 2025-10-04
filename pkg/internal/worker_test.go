package internal

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		backoffLimit  int
		sleep         time.Duration
		function      func() error
		expectError   bool
		errorContains string
	}{
		{
			name:         "Function succeeds on first attempt",
			backoffLimit: 3,
			sleep:        10 * time.Millisecond,
			function: func() error {
				return nil
			},
			expectError: false,
		},
		{
			name:         "Function succeeds on second attempt",
			backoffLimit: 3,
			sleep:        10 * time.Millisecond,
			function: func() func() error {
				attempt := 0
				return func() error {
					attempt++
					if attempt < 2 {
						return errors.New("temporary error")
					}
					return nil
				}
			}(),
			expectError: false,
		},
		{
			name:         "Function succeeds on third attempt",
			backoffLimit: 3,
			sleep:        10 * time.Millisecond,
			function: func() func() error {
				attempt := 0
				return func() error {
					attempt++
					if attempt < 3 {
						return errors.New("temporary error")
					}
					return nil
				}
			}(),
			expectError: false,
		},
		{
			name:         "Function fails all attempts",
			backoffLimit: 3,
			sleep:        10 * time.Millisecond,
			function: func() error {
				return errors.New("persistent error")
			},
			expectError:   true,
			errorContains: "retry failed after 3 attempts",
		},
		{
			name:         "Function fails with specific error preserved",
			backoffLimit: 2,
			sleep:        10 * time.Millisecond,
			function: func() error {
				return errors.New("database connection failed")
			},
			expectError:   true,
			errorContains: "database connection failed",
		},
		{
			name:         "Zero backoff limit",
			backoffLimit: 0,
			sleep:        10 * time.Millisecond,
			function: func() error {
				return errors.New("should fail immediately")
			},
			expectError:   true,
			errorContains: "retry failed after 0 attempts",
		},
		{
			name:         "Single backoff limit with success",
			backoffLimit: 1,
			sleep:        10 * time.Millisecond,
			function: func() error {
				return nil
			},
			expectError: false,
		},
		{
			name:         "Single backoff limit with failure",
			backoffLimit: 1,
			sleep:        10 * time.Millisecond,
			function: func() error {
				return errors.New("immediate failure")
			},
			expectError:   true,
			errorContains: "retry failed after 1 attempts",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := Retry(tc.backoffLimit, tc.sleep, tc.function)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tc.expectError && tc.errorContains != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain '%s', got: %v", tc.errorContains, err)
				}
			}
		})
	}
}
