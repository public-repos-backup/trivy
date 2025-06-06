package module_test

import (
	"io/fs"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aquasecurity/trivy/pkg/extension"
	"github.com/aquasecurity/trivy/pkg/fanal/analyzer"
	"github.com/aquasecurity/trivy/pkg/module"
)

func TestManager_Register(t *testing.T) {
	if runtime.GOOS == "windows" {
		// WASM tests difficult on Windows
		t.Skip("Test satisfied adequately by Linux tests")
	}
	tests := []struct {
		name                 string
		moduleDir            string
		enabledModules       []string
		wantAnalyzerVersions analyzer.Versions
		wantExtentions       []string
		wantErr              bool
	}{
		{
			name:      "happy path",
			moduleDir: "testdata/happy",
			wantAnalyzerVersions: analyzer.Versions{
				Analyzers: map[string]int{
					"happy": 1,
				},
				PostAnalyzers: make(map[string]int),
			},
			wantExtentions: []string{
				"happy",
			},
		},
		{
			name:      "only analyzer",
			moduleDir: "testdata/analyzer",
			wantAnalyzerVersions: analyzer.Versions{
				Analyzers: map[string]int{
					"analyzer": 1,
				},
				PostAnalyzers: make(map[string]int),
			},
			wantExtentions: []string{},
		},
		{
			name:      "only post scanner",
			moduleDir: "testdata/scanner",
			wantAnalyzerVersions: analyzer.Versions{
				Analyzers:     make(map[string]int),
				PostAnalyzers: make(map[string]int),
			},
			wantExtentions: []string{
				"scanner",
			},
		},
		{
			name:      "no module dir",
			moduleDir: "no-such-dir",
			wantAnalyzerVersions: analyzer.Versions{
				Analyzers:     make(map[string]int),
				PostAnalyzers: make(map[string]int),
			},
			wantExtentions: []string{},
		},
		{
			name:      "pass enabled modules",
			moduleDir: "testdata",
			enabledModules: []string{
				"happy",
				"analyzer",
			},
			wantAnalyzerVersions: analyzer.Versions{
				Analyzers: map[string]int{
					"happy":    1,
					"analyzer": 1,
				},
				PostAnalyzers: make(map[string]int),
			},
			wantExtentions: []string{
				"happy",
			},
		},
	}

	// Confirm that wasm modules are generated beforehand
	var count int
	err := filepath.WalkDir("testdata", func(path string, _ fs.DirEntry, _ error) error {
		if filepath.Ext(path) == ".wasm" {
			count++
		}
		return nil
	})
	require.NoError(t, err)
	// WASM modules must be generated before running the tests.
	require.Equal(t, 3, count, "missing WASM modules, try 'mage test:unit' or 'mage test:generateModules'")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := module.NewManager(t.Context(), module.Options{
				Dir:            tt.moduleDir,
				EnabledModules: tt.enabledModules,
			})
			require.NoError(t, err)

			// Register analyzer and post scanner from WASM module
			m.Register()

			// Remove registered analyzers and post scanners so that it will not affect other tests.
			defer m.Deregister()

			// Confirm the analyzer is registered
			a, err := analyzer.NewAnalyzerGroup(analyzer.AnalyzerOptions{})
			require.NoError(t, err)

			got := a.AnalyzerVersions()
			assert.Equal(t, tt.wantAnalyzerVersions, got)

			hookNames := lo.Map(extension.Hooks(), func(hook extension.Hook, _ int) string {
				return hook.Name()
			})
			assert.Equal(t, tt.wantExtentions, hookNames)
		})
	}
}
