package packagist

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersions(t *testing.T) {
	data, err := os.ReadFile("testdata/psr_log.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/p2/psr/log.json", r.URL.Path)
		w.Write(data)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	versions, err := c.Versions("psr/log")
	require.NoError(t, err)
	require.Len(t, versions, 3)

	assert.Equal(t, "3.0.2", versions[0].Version)
	assert.Equal(t, "git", versions[0].Source.Type)
	assert.Equal(t, "https://github.com/php-fig/log.git", versions[0].Source.URL)
	assert.Equal(t, "f16e1d5863e37f8d8c2a01719f5b34baa2b714d3", versions[0].Source.Reference)
	assert.Equal(t, "zip", versions[0].Dist.Type)

	assert.Equal(t, "1.1.4", versions[2].Version)
	assert.Equal(t, "d49695b909c3b7628b6289db5479a1c204601f11", versions[2].Source.Reference)
}

func TestVersions_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Versions("nonexistent/pkg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestVersions_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{invalid json"))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := c.Versions("psr/log")
	require.Error(t, err)
}

func TestVersions_InvalidPackageName(t *testing.T) {
	c := &Client{}
	_, err := c.Versions("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vendor/package")
}
