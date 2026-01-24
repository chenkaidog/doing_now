package encode

import "testing"

func TestEncodePassword(t *testing.T) {
	t.Log(EncodePassword("salt", "123456"))
}
