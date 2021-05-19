package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/nacl/box"
)

func Encrypt(value int, r []byte, pk *[32]byte) (c []byte, err error) {
	// verify size of r
	if len(r) != RSize {
		return nil, fmt.Errorf("invalid randomness size")
	}
	// convert value to bytes
	valueBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(valueBytes, uint32(value))
	// concatenate value bytes to randomness
	msg := append(valueBytes, r...)
	// encrypt using X25519 key exchange and Salsa20/Poly1305
	return box.SealAnonymous(nil, msg, pk, rand.Reader)
}

func Decrypt(c []byte, pk, sk *[32]byte) (value int, r []byte, err error) {
	msg, ok := box.OpenAnonymous(nil, c, pk, sk)
	if !ok || len(msg) < RSize + 4 {
		return 0, nil, fmt.Errorf("decryption failed")
	}
	valueBytes := msg[:4]
	value = int(binary.LittleEndian.Uint32(valueBytes))
	r = msg[4:]
	return
}