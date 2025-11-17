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
	"github.com/stretchr/testify/require"
)

func TestHelmRepoAddIntegration(t *testing.T) {
	helmPath, available := isHelmAvailable(t)
	require.True(t, available, "helm binary not found in PATH")

	if util.ExecutablePaths == nil {
		util.ExecutablePaths = make(map[string]string)
	}
	util.ExecutablePaths["helm"] = helmPath

	tests := []struct {
		name           string
		config         *ConfigurationSpecs
		skipCleanup    bool
		expectFatal    bool
		validateResult func(*testing.T, string)
	}{
		{
			name: "Successfully add public helm repository",
			config: &ConfigurationSpecs{
				Configuration: Configuration{
					HelmChartConfiguration: HelmChartConfiguration{
						RepoAlias: "kubeslice-itest-success",
						RepoUrl:   "https://kubeslice.github.io/kubeslice/",
						UseLocal:  false,
					},
				},
			},
			expectFatal: false,
			validateResult: func(t *testing.T, repoAlias string) {
				output := getHelmRepoList(t)
				require.Contains(t, output, repoAlias,
					"helm repo list should contain the added repository")
				require.Contains(t, output, "https://kubeslice.github.io/kubeslice/",
					"helm repo list should contain the correct URL")
			},
		},
		{
			name: "UseLocal flag skips helm operations",
			config: &ConfigurationSpecs{
				Configuration: Configuration{
					HelmChartConfiguration: HelmChartConfiguration{
						RepoAlias: "kubeslice-itest-skip",
						RepoUrl:   "https://example.com/charts",
						UseLocal:  true,
					},
				},
			},
			skipCleanup: true,
			expectFatal: false,
			validateResult: func(t *testing.T, repoAlias string) {
				output := getHelmRepoList(t)
				require.NotContains(t, output, repoAlias,
					"Repository should not be added when UseLocal is true")
			},
		},
		{
			name: "Fails to add non-existent repository",
			config: &ConfigurationSpecs{
				Configuration: Configuration{
					HelmChartConfiguration: HelmChartConfiguration{
						RepoAlias: "kubeslice-itest-fail",
						RepoUrl:   "https://non-existent-repo-12345.example.com/charts",
						UseLocal:  false,
					},
				},
			},
			skipCleanup: true,
			expectFatal: true,
			validateResult: func(t *testing.T, repoAlias string) {
				output := getHelmRepoList(t)
				require.NotContains(t, output, repoAlias,
					"Failed test should not add the repository")
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

			if tt.expectFatal {
				require.NotEmpty(t, fakeOutput.FatalCalls,
					"Expected AddHelmCharts to call Fatalf for failed repo add")
				assert.Contains(t, fakeOutput.FatalCalls[0], "Process failed")
			} else {
				require.Empty(t, fakeOutput.FatalCalls,
					"AddHelmCharts should not have called Fatalf. Errors: %v", fakeOutput.FatalCalls)
			}

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
		t.Logf("helm repo list error: %v, stderr: %s", err, stderr.String())
		return ""
	}
	return stdout.String()
}

func cleanupHelmRepo(t *testing.T, repoAlias string) {
	t.Helper()
	output := getHelmRepoList(t)
	if !strings.Contains(output, repoAlias) {
		return
	}

	cmd := exec.Command(util.ExecutablePaths["helm"], "repo", "remove", repoAlias)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Logf("Failed to cleanup helm repo %s: %v", repoAlias, err)
	}
}
