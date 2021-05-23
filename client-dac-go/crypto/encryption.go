package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/nacl/box"
	"math/big"
)

func Encrypt(value int, r *big.Int, pk *[32]byte) (c []byte, err error) {
	// convert value to bytes
	msg := make([]byte, 36)
	binary.LittleEndian.PutUint32(msg[:4], uint32(value))
	// concatenate value bytes to randomness
	r.FillBytes(msg[4:])
	// encrypt using X25519 key exchange and Salsa20/Poly1305
	return box.SealAnonymous(nil, msg, pk, rand.Reader)
}

func Decrypt(c []byte, pk, sk *[32]byte) (value int, r *big.Int, err error) {
	msg, ok := box.OpenAnonymous(nil, c, pk, sk)
	if !ok || len(msg) != 36 {
		return 0, nil, fmt.Errorf("decryption failed")
	}
	valueBytes := msg[:4]
	value = int(binary.LittleEndian.Uint32(valueBytes))
	r = new(big.Int)
	r.SetBytes(msg[4:])
	return
}