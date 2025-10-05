package internal

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenerateKubeSliceProjectYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		projectName   string
		users         []string
		expectedUsers string
	}{
		{
			name:        "With multiple users",
			projectName: "test-project",
			users:       []string{"alice", "bob"},
			expectedUsers: `
      - alice
      - bob
`,
		},
		{
			name:        "With a single user",
			projectName: "prod-project",
			users:       []string{"charlie"},
			expectedUsers: `
      - charlie
`,
		},
		{
			name:        "With no users provided, defaults to admin",
			projectName: "default-project",
			users:       []string{},
			expectedUsers: `
      - admin
`,
		},
		{
			name:        "With nil users slice, defaults to admin",
			projectName: "nil-project",
			users:       nil,
			expectedUsers: `
      - admin
`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := generateKubeSliceProjectYAML(tc.projectName, tc.users)
			expected := fmt.Sprintf(kubesliceProjectTemplate, tc.projectName, tc.expectedUsers)
			if strings.TrimSpace(got) != strings.TrimSpace(expected) {
				t.Errorf("generateKubeSliceProjectYAML() mismatch:\n\nwant:\n%s\n\ngot:\n%s", expected, got)
			}
		})
	}
}
