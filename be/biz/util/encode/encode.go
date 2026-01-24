package encode

import (
	"crypto/sha256"
	"encoding/hex"
)

func EncodePassword(salt, password string) string {
	h := sha256.New()

	h.Write([]byte(salt))
	h.Write([]byte(password))

	return hex.EncodeToString(h.Sum(nil))
}
