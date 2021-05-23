package dacidentity

import (
	"errors"
	"fmt"
	"github.com/dbogatov/dac-lib/dac"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/cryptosuite"
)

type SigningManager struct {
	cryptoProvider core.CryptoSuite
	hashOpts       core.HashOpts
}

// New Constructor for a signing manager.
// @param {BCCSP} cryptoProvider - crypto provider
// @param {Config} config - configuration provider
// @returns {SigningManager} new signing manager
func New(cryptoProvider core.CryptoSuite) (*SigningManager, error) {
	return &SigningManager{cryptoProvider: cryptoProvider, hashOpts: cryptosuite.GetSHAOpts()}, nil
}

// Sign will sign the given object using provided key
func (mgr *SigningManager) Sign(object []byte, key core.Key) ([]byte, error) {

	if len(object) == 0 {
		return nil, errors.New("object (to sign) required")
	}

	if key == nil {
		return nil, errors.New("key (for signing) required")
	}

	digest, err := mgr.cryptoProvider.Hash(object, mgr.hashOpts)
	if err != nil {
		return nil, err
	}
	prg := NewRand()
	nymKey, ok := key.(NymKey)
	if !ok {
		return nil, errors.New("Key must be a NymKey")
	}
	fmt.Println(digest)
	fmt.Println(dac.PointToBytes(nymKey.PublicNymKey()))
	signature := dac.SignNym(prg, nymKey.PublicNymKey(), nymKey.PrivateNymKey(), nymKey.PrivateKey(), nymKey.H(), digest)
	return signature.ToBytes(), nil
}
