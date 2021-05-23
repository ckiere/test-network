package dacidentity

import (
	"fmt"
	"github.com/golang/protobuf/proto"

	"github.com/dbogatov/dac-lib/dac"
	pb_msp "github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/pkg/errors"
)

// User is a representation of a Fabric user
type User struct {
	id        string
	mspID     string
	creds     dac.Credentials
	sk        dac.SK
	tempProof dac.Proof
	nymKey    NymKey
	H         interface{}
	Ys        [][]interface{}
	RootPk    interface{}
}

// Create user from configuration
func CreateUser(dacConfig DacConfig, credConfig CredentialsConfig, id string, mspID string) (*User, error) {
	ys, err := dacConfig.Ys()
	if err != nil {
		return nil, err
	}
	rootPk, err := dacConfig.RootPk()
	if err != nil {
		return nil, err
	}
	h, err := dacConfig.H()
	if err != nil {
		return nil, err
	}
	user := &User{
		id: id,
		mspID: mspID,
		creds: *credConfig.Credentials(),
		sk: credConfig.Sk(),
		H: h,
		Ys: ys,
		RootPk: rootPk,
	}
	user.UpdateNymIdentity()
	return user, nil
}

// Identifier returns user identifier
func (u *User) Identifier() *msp.IdentityIdentifier {
	return &msp.IdentityIdentifier{MSPID: u.mspID, ID: u.id}
}

// Verify a signature over some message using this identity as reference
func (u *User) Verify(msg []byte, sig []byte) error {
	return errors.New("not implemented")
}

// Serialize converts an identity to bytes
func (u *User) Serialize() ([]byte, error) {
	// Serialize the identity
	nymBytes := dac.PointToBytes(u.nymKey.PublicNymKey())
	serializedDacIdentity := &pb_msp.SerializedIdemixIdentity{
		NymX:  nymBytes[:len(nymBytes)/2],
		NymY:  nymBytes[len(nymBytes)/2:],
		Proof: u.tempProof.ToBytes(),
	}
	dacIdentityBytes, err := proto.Marshal(serializedDacIdentity)
	if err != nil {
		return nil, errors.Wrap(err, "marshal serializedDacIdentity failed")
	}
	serializedIdentity := &pb_msp.SerializedIdentity{
		Mspid:   u.mspID,
		IdBytes: dacIdentityBytes,
	}
	identityBytes, err := proto.Marshal(serializedIdentity)
	if err != nil {
		return nil, errors.Wrap(err, "marshal serializedIdentity failed")
	}
	return identityBytes, nil
}

// EnrollmentCertificate Returns the underlying ECert representing this user’s identity.
func (u *User) EnrollmentCertificate() []byte {
	return nil
}

// PrivateKey returns the crypto suite representation of the private key
func (u *User) PrivateKey() core.Key {
	return u.nymKey
}

// PublicVersion returns the public parts of this identity
func (u *User) PublicVersion() msp.Identity {
	return nil
}

// Sign the message
func (u *User) Sign(msg []byte) ([]byte, error) {
	return nil, errors.New("Sign() function not implemented")
}

func (u *User) UpdateNymIdentity() {

	prg := NewRand()

	skNym, pkNym := dac.GenerateNymKeys(prg, u.sk, u.H)
	indices := dac.Indices{}

	proof, e := u.creds.Prove(
		prg,
		u.sk,
		u.RootPk,
		indices,
		[]byte{},
		u.Ys,
		u.H,
		skNym,
	)

	if e != nil {
		panic("Failed to generate credential proof")
	}

	u.nymKey = NymKey{privateKey: u.sk, privateNymKey: skNym, publicNymKey: pkNym, h: u.H}
	u.tempProof = proof
	fmt.Println("Nym key updated")
}

type NymKey struct {
	privateKey    dac.SK
	privateNymKey dac.SK
	publicNymKey  interface{}
	h             interface{}
}

func (n NymKey) Bytes() ([]byte, error) {
	panic("not implemented")
}

func (n NymKey) SKI() []byte {
	panic("not implemented")
}

func (n NymKey) Symmetric() bool {
	return false
}

func (n NymKey) Private() bool {
	return n.privateKey != nil
}

func (n NymKey) PrivateKey() dac.SK {
	return n.privateKey
}

func (n NymKey) PrivateNymKey() dac.SK {
	return n.privateNymKey
}

func (n NymKey) PublicNymKey() interface{} {
	return n.publicNymKey
}

func (n NymKey) PublicKey() (core.Key, error) {
	panic("not implemented")
}

func (n NymKey) H() interface{} {
	return n.h
}
