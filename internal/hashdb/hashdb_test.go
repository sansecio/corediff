package hashdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	db, err := Load("../../fixture/sample.db")
	require.NoError(t, err)
	assert.Len(t, db, 238)
	wantHash := uint64(3900178074848893275)
	assert.True(t, db.Contains(wantHash))
}
