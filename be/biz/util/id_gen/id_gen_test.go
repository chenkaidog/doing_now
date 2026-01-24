package id_gen

import (
	"testing"
	"time"
)

func TestNewId(t *testing.T) {
	idgen := NewIDGenerator(2)
	ticker := time.NewTicker(time.Millisecond)

	for i := 0; ; i++ {
		select {
		case <-ticker.C:
			t.Logf("log id: %s", idgen.NewID())
			if i > 10 {
				idgen.Stop()
			}
		case <-idgen.stop:
			return
		}
	}
}
