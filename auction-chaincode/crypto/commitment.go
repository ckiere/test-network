package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
)

func Commit(value int) ([]byte, []byte) {
	// get randomness from cryptographically secure generator
	r := make([]byte, 4)
	_, err := rand.Read(r)
	if err != nil {
		panic(err)
	}
	// compute hash commitment
	h := sha256.New()
	h.Write(r)
	valueBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(valueBytes, uint32(value))
	h.Write(valueBytes)
	return h.Sum(nil), r
}

func CheckCommit(com []byte, value int, r []byte) bool {
	// check params sanity
	if len(com) != 32 || len(r) != 4 {
		return false
	}
	// compute hash commitment
	h := sha256.New()
	h.Write(r)
	valueBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(valueBytes, uint32(value))
	h.Write(valueBytes)
	return bytes.Equal(com, h.Sum(nil))
}