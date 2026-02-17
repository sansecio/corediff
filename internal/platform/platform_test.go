package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		files    []string // relative paths to create
		wantName string   // expected platform Name, "" for nil
	}{
		{
			name:     "magento2 with env.php",
			files:    []string{"app/etc/env.php"},
			wantName: "magento2",
		},
		{
			name:     "magento2 with lib/internal/Magento",
			files:    []string{"lib/internal/Magento/.gitkeep"},
			wantName: "magento2",
		},
		{
			name:     "magento2 with app/design/frontend/Magento",
			files:    []string{"app/design/frontend/Magento/.gitkeep"},
			wantName: "magento2",
		},
		{
			name:     "magento1",
			files:    []string{"app/etc/local.xml"},
			wantName: "magento1",
		},
		{
			name:     "wordpress",
			files:    []string{"wp-config.php"},
			wantName: "wordpress",
		},
		{
			name:     "magento2 takes priority over magento1",
			files:    []string{"app/etc/env.php", "app/etc/local.xml"},
			wantName: "magento2",
		},
		{
			name:     "unknown platform",
			files:    []string{"index.html"},
			wantName: "",
		},
		{
			name:     "empty dir",
			files:    nil,
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for _, f := range tt.files {
				abs := filepath.Join(root, f)
				require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
				require.NoError(t, os.WriteFile(abs, nil, 0o644))
			}

			p := Detect(root)
			if tt.wantName == "" {
				assert.Nil(t, p)
			} else {
				require.NotNil(t, p)
				assert.Equal(t, tt.wantName, p.Name)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	tests := []struct {
		platform *Platform
		path     string
		want     bool
	}{
		{Magento2, "var/cache/foo.php", true},
		{Magento2, "var/log/system.log", true},
		{Magento2, "vendor/composer/autoload_real.php", true},
		{Magento2, "vendor/composer/autoload_classmap.php", true},
		{Magento2, "vendor/magento/module-catalog/Model/Product.php", false},
		{Magento2, "app/code/Vendor/Module/Model.php", false},
		{Magento2, "generated/code/Magento/Foo/Interceptor.php", false}, // NOT excluded — handled by validator
		{Magento1, "app/code/local/Foo.php", false},
		{WordPress, "wp-content/plugins/foo.php", false},
	}

	for _, tt := range tests {
		t.Run(tt.platform.Name+"/"+tt.path, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.platform.IsExcluded(tt.path))
		})
	}
}
