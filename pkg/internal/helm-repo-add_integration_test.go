//go:build integration
// +build integration

package internal

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/kubeslice/kubeslice-cli/util"
	"github.com/stretchr/testify/assert"
)

// TestHelmRepoAddIntegration verifies that AddHelmCharts can successfully communicate with the real helm binary while safely handling any util.Fatalf calls.
func TestHelmRepoAddIntegration(t *testing.T) {
	helmPath, available := isHelmAvailable(t)
	if !available {
		t.Skip("Skipping: helm binary not found in PATH")
	}

	if util.ExecutablePaths == nil {
		util.ExecutablePaths = make(map[string]string)
	}
	util.ExecutablePaths["helm"] = helmPath

	tests := []struct {
		name           string
		config         *ConfigurationSpecs
		skipCleanup    bool
		validateResult func(*testing.T, string)
	}{
		{
			name: "Successfully add public helm repository",
			config: &ConfigurationSpecs{
				Configuration: Configuration{
					HelmChartConfiguration: HelmChartConfiguration{
						RepoAlias: "kubeslice-itest-1",
						RepoUrl:   "https://kubeslice.github.io/kubeslice/",
						UseLocal:  false,
					},
				},
			},
			validateResult: func(t *testing.T, repoAlias string) {
				output := getHelmRepoList(t)
				assert.Contains(t, output, repoAlias,
					"helm repo list should contain the added repository")
				assert.Contains(t, output, "https://kubeslice.github.io/kubeslice/",
					"helm repo list should contain the correct URL")
			},
		},
		{
			name: "UseLocal flag skips helm operations",
			config: &ConfigurationSpecs{
				Configuration: Configuration{
					HelmChartConfiguration: HelmChartConfiguration{
						RepoAlias: "should-not-be-added",
						RepoUrl:   "https://example.com/charts",
						UseLocal:  true,
					},
				},
			},
			skipCleanup: true,
			validateResult: func(t *testing.T, repoAlias string) {
				output := getHelmRepoList(t)
				assert.NotContains(t, output, repoAlias,
					"Repository should not be added when UseLocal is true")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoAlias := tt.config.Configuration.HelmChartConfiguration.RepoAlias

			originalExecutor := util.CommandExecutor

			cleanupEnv := util.NewTestEnvironment()
			util.CommandExecutor = originalExecutor

			fakeOutput := util.Output.(*util.FakeOutput)
			defer cleanupEnv()

			if !tt.skipCleanup {
				t.Cleanup(func() {
					cleanupHelmRepo(t, repoAlias)
				})
			}

			AddHelmCharts(tt.config)

			assert.Empty(t, fakeOutput.FatalCalls,
				"AddHelmCharts should not have called Fatalf. Errors: %v", fakeOutput.FatalCalls)

			if tt.validateResult != nil {
				tt.validateResult(t, repoAlias)
			}
		})
	}
}

func isHelmAvailable(t *testing.T) (string, bool) {
	t.Helper()
	path, err := exec.LookPath("helm")
	return path, err == nil
}

func getHelmRepoList(t *testing.T) string {
	t.Helper()
	cmd := exec.Command(util.ExecutablePaths["helm"], "repo", "list")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if strings.Contains(stderr.String(), "no repositories") {
			return ""
		}
		t.Logf("helm repo list returned error: %v, stderr: %s", err, stderr.String())
		return ""
	}
	return stdout.String()
}

func cleanupHelmRepo(t *testing.T, repoAlias string) {
	t.Helper()
	output := getHelmRepoList(t)
	if !strings.Contains(output, repoAlias) {
		t.Logf("Repository %s not found, skipping cleanup", repoAlias)
		return
	}

	cmd := exec.Command(util.ExecutablePaths["helm"], "repo", "remove", repoAlias)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to cleanup helm repo %s: %v, stderr: %s",
			repoAlias, err, stderr.String())
	} else {
		t.Logf("Successfully cleaned up helm repo: %s", repoAlias)
	}
}
