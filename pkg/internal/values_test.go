package internal

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestMergeMaps(t *testing.T) {
	tests := []struct {
		name     string
		dest     map[interface{}]interface{}
		src      map[interface{}]interface{}
		expected map[interface{}]interface{}
	}{
		{
			name:     "simple merge with no conflicts",
			dest:     map[interface{}]interface{}{"a": 1},
			src:      map[interface{}]interface{}{"b": 2},
			expected: map[interface{}]interface{}{"a": 1, "b": 2},
		},
		{
			name:     "source overwrites destination value",
			dest:     map[interface{}]interface{}{"a": 1},
			src:      map[interface{}]interface{}{"a": 2},
			expected: map[interface{}]interface{}{"a": 2},
		},
		{
			name:     "deep merge of nested maps",
			dest:     map[interface{}]interface{}{"a": map[interface{}]interface{}{"b": 1, "c": 2}},
			src:      map[interface{}]interface{}{"a": map[interface{}]interface{}{"b": 99, "d": 4}},
			expected: map[interface{}]interface{}{"a": map[interface{}]interface{}{"b": 99, "c": 2, "d": 4}},
		},
		{
			name:     "source map overwrites destination non-map",
			dest:     map[interface{}]interface{}{"a": 1},
			src:      map[interface{}]interface{}{"a": map[interface{}]interface{}{"b": 2}},
			expected: map[interface{}]interface{}{"a": map[interface{}]interface{}{"b": 2}},
		},
		{
			name:     "empty source map does not change destination",
			dest:     map[interface{}]interface{}{"a": 1},
			src:      map[interface{}]interface{}{},
			expected: map[interface{}]interface{}{"a": 1},
		},
		{
			name:     "empty destination map becomes source",
			dest:     map[interface{}]interface{}{},
			src:      map[interface{}]interface{}{"a": 1},
			expected: map[interface{}]interface{}{"a": 1},
		},
		{
			name:     "both maps empty returns empty map",
			dest:     map[interface{}]interface{}{},
			src:      map[interface{}]interface{}{},
			expected: map[interface{}]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeMaps(tt.dest, tt.src)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("mergeMaps() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateValuesFile(t *testing.T) {
	tests := []struct {
		name         string
		hc           *HelmChart
		defaults     string
		expectError  bool
		expectedYAML string
	}{
		{
			name: "successful merge of values and defaults",
			hc: &HelmChart{
				Values: map[string]interface{}{
					"controller.logLevel": "debug",
					"image.tag":           "latest",
				},
			},
			defaults: `
controller:
  logLevel: "info" # This should be overwritten
  replicas: 1
image:
  repository: "nginx"
`,
			expectError: false,
			expectedYAML: `
controller:
  logLevel: debug
  replicas: 1
image:
  repository: nginx
  tag: latest
`,
		},
		{
			name: "only helm chart values, no defaults",
			hc: &HelmChart{
				Values: map[string]interface{}{
					"service.type": "ClusterIP",
					"service.port": 8080,
				},
			},
			defaults:    "",
			expectError: false,
			expectedYAML: `
service:
  port: 8080
  type: ClusterIP
`,
		},
		{
			name: "only defaults, no helm chart values",
			hc: &HelmChart{
				Values: nil,
			},
			defaults: `
global:
  clusterName: "test-cluster"
`,
			expectError: false,
			expectedYAML: `
global:
  clusterName: test-cluster
`,
		},
		{
			name:        "invalid defaults YAML should return an error",
			hc:          &HelmChart{},
			defaults:    "invalid: yaml: content: [unclosed",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "values.yaml")

			err := generateValuesFile(filePath, tt.hc, tt.defaults)

			if (err != nil) != tt.expectError {
				t.Fatalf("generateValuesFile() error = %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError {
				content, readErr := ioutil.ReadFile(filePath)
				if readErr != nil {
					t.Fatalf("Failed to read generated file: %v", readErr)
				}
				var actualMap, expectedMap map[interface{}]interface{}
				if err := yaml.Unmarshal(content, &actualMap); err != nil {
					t.Fatalf("Failed to unmarshal actual YAML: %v", err)
				}
				if err := yaml.Unmarshal([]byte(tt.expectedYAML), &expectedMap); err != nil {
					t.Fatalf("Failed to unmarshal expected YAML: %v", err)
				}
				if !reflect.DeepEqual(actualMap, expectedMap) {
					t.Errorf("Generated YAML content mismatch.\nGot:\n%s\nWant:\n%s", string(content), tt.expectedYAML)
				}
			}
		})
	}
}

func TestGenerateValuesFile_EdgeCases(t *testing.T) {
	t.Run("write error to read-only directory", func(t *testing.T) {
		tempDir := t.TempDir()
		readOnlyPath := filepath.Join(tempDir, "read-only-dir")
		if err := os.Mkdir(readOnlyPath, 0555); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}

		filePath := filepath.Join(readOnlyPath, "values.yaml")
		hc := &HelmChart{Values: map[string]interface{}{"a": "b"}}

		err := generateValuesFile(filePath, hc, "")

		if err == nil {
			t.Error("Expected an error when writing to a read-only directory, but got nil")
		}
	})

	t.Run("nil helm chart struct should be handled gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "values.yaml")
		defaults := "defaultKey: defaultValue"
		expectedYAML := "defaultKey: defaultValue\n"

		err := generateValuesFile(filePath, nil, defaults)
		if err != nil {
			t.Fatalf("generateValuesFile() returned an unexpected error for a nil HelmChart: %v", err)
		}

		content, readErr := ioutil.ReadFile(filePath)
		if readErr != nil {
			t.Fatalf("Failed to read generated file: %v", readErr)
		}
		if string(content) != expectedYAML {
			t.Errorf("Expected file with only defaults, but content differs.\nGot:\n%s\nWant:\n%s", string(content), expectedYAML)
		}
	})
}
