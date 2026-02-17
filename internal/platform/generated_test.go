package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMagentoGenerated_HandledPaths(t *testing.T) {
	tests := []struct {
		relPath string
		handled bool
	}{
		{"generated/code/Magento/Foo/Interceptor.php", true},
		{"generated/code/Magento/Foo/Factory.php", true},
		{"generated/code/Magento/Foo/Proxy.php", true},
		{"generated/code/Magento/Foo/Extension.php", true},
		{"generated/code/Magento/Foo/ExtensionInterface.php", true},
		{"generated/code/Magento/Foo/SomeFactory.php", true},           // *Factory.php
		{"generated/code/Magento/Foo/SomeInterfaceFactory.php", true},  // *Factory.php
		{"generated/code/Magento/Foo/Model.php", false},                // not a known generated type
		{"generated/code/Magento/Foo/Helper.php", false},               // not a known generated type
		{"generated/metadata/foo.php", false},                          // not under generated/code/
		{"vendor/magento/module-foo/Interceptor.php", false},           // not under generated/
		{"app/code/Vendor/Module/Interceptor.php", false},              // not under generated/
		{"generated/code/DOMDocumentFactory.php", true},                // Factory at root level
	}

	scanBuf := make([]byte, 1024)
	for _, tt := range tests {
		t.Run(tt.relPath, func(t *testing.T) {
			handled, _, _ := validateMagentoGenerated(tt.relPath, "/dummy/"+tt.relPath, scanBuf)
			assert.Equal(t, tt.handled, handled, "handled mismatch for %s", tt.relPath)
		})
	}
}

func TestValidateMagentoGenerated_FixtureFiles(t *testing.T) {
	// testdata/ contains a curated subset of generated files covering all types
	// and edge cases (exec method, _resetState, saveFileToFilesystem, no-namespace
	// factory, empty/full extensions).
	fixtureRoot := filepath.Join("testdata", "generated", "code")

	scanBuf := make([]byte, 1024*1024)
	var checked int

	err := filepath.Walk(fixtureRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".php") {
			return nil
		}

		relPath, relErr := filepath.Rel("testdata", path)
		require.NoError(t, relErr)

		handled, hits, lines := validateMagentoGenerated(relPath, path, scanBuf)
		if !handled {
			return nil
		}
		checked++

		if len(hits) > 0 {
			var flagged []string
			for i, lineNo := range hits {
				flagged = append(flagged, fmt.Sprintf("%s (line %d)",
					strings.TrimSpace(string(lines[i])), lineNo))
			}
			t.Errorf("false positive in %s:\n  %s", relPath, strings.Join(flagged, "\n  "))
		}

		return nil
	})
	require.NoError(t, err)
	t.Logf("validated %d generated fixture files with zero false positives", checked)
	assert.Equal(t, 11, checked, "expected to check all 11 testdata fixture files")
}

func TestValidateMagentoGenerated_DetectsMalware(t *testing.T) {
	maliciousLines := []string{
		`eval($decoded);`,
		`$x = base64_decode($payload);`,
		`gzinflate($data);`,
		`shell_exec('rm -rf /');`,
		`system($cmd);`,
		`exec($command);`,
		`file_put_contents('/tmp/shell.php', $code);`,
		`$input = $_GET['cmd'];`,
		`$data = $_POST['payload'];`,
		`$r = $_REQUEST['x'];`,
	}

	for _, mal := range maliciousLines {
		t.Run(mal, func(t *testing.T) {
			content := "<?php\nnamespace Magento\\Foo;\nclass FooInterceptor extends \\Magento\\Foo implements \\Magento\\Framework\\Interception\\InterceptorInterface\n{\n" + mal + "\n}\n"
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "Interceptor.php")
			require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

			scanBuf := make([]byte, 1024*1024)
			handled, hits, lines := validateMagentoGenerated("generated/code/Magento/Foo/Interceptor.php", tmpFile, scanBuf)
			assert.True(t, handled)
			assert.NotEmpty(t, hits, "expected malicious line to be flagged")
			assert.NotEmpty(t, lines)
		})
	}
}
