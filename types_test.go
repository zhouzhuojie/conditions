package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInspectDataType(t *testing.T) {
	assert.Equal(t, Number, InspectDataType(float64(1.0)))
	assert.Equal(t, Boolean, InspectDataType(true))
	assert.Equal(t, String, InspectDataType("hello"))
	assert.Equal(t, Unknown, InspectDataType(42))
	assert.Equal(t, Unknown, InspectDataType(nil))
}
