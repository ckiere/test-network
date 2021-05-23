package dacidentity

import (
	"github.com/dbogatov/fabric-amcl/amcl"
	"math/rand"
	"sync"
	"time"
)

var randMutex = &sync.Mutex{}

// NewRand ...
func NewRand() (prg *amcl.RAND) {

	randMutex.Lock()
	defer randMutex.Unlock()

	prg = amcl.NewRAND()
	goPrg := rand.New(rand.NewSource(time.Now().UnixNano()))
	var raw [32]byte
	for i := 0; i < 32; i++ {
		raw[i] = byte(goPrg.Int())
	}
	prg.Seed(32, raw[:])

	return
}

// NewRandSeed ...
func NewRandSeed(seed []byte) (prg *amcl.RAND) {

	prg = amcl.NewRAND()
	prg.Seed(len(seed), seed)

	return
}
