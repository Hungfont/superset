package query

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnsupportedEstimatorReturnsSupportedFalse(t *testing.T) {
	e := &UnsupportedEstimator{driver: "sqlite"}
	result, err := e.Estimate(context.Background(), "SELECT 1", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Supported)
}

func TestUnsupportedEstimatorNoPanicWithNilConn(t *testing.T) {
	e := &UnsupportedEstimator{driver: "mongodb"}
	result, err := e.Estimate(context.Background(), "db.collection.find()", nil)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Supported)
}
