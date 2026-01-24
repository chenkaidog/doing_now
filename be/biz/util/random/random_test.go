package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandStr(t *testing.T) {
	for i := 0; i <= 10; i++ {
		s := RandStr(i)
		t.Logf("rand str: %s", s)
		assert.Equal(t, i, len(s))
	}
}
