package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/twistededwards"
	"math/big"
)

var curveParams = twistededwards.GetEdwardsCurve()
var order = curveParams.Order
var hx = new(fr.Element).SetString("51295569138718539371092613972351202357326289069440880621285444911501458459494")
var hy = new(fr.Element).SetString("49831129265363587078046764490824666482509464638593900758877649985443393819454")
var h = twistededwards.NewPointAffine(*hx, *hy)

func Commit(value int) (*twistededwards.PointAffine, *big.Int, error) {
	p := twistededwards.PointAffine{}
	i := big.NewInt(int64(value))
	r, err := Random()
	if err != nil {
		return nil, nil, err
	}
	p.ScalarMul(&curveParams.Base, i)
	temp := twistededwards.PointAffine{}
	temp.ScalarMul(&h, r)
	p.Add(&p, &temp)
	return &p, r, nil

}

func CheckCommit(value int, r *big.Int, com *twistededwards.PointAffine) bool {
	if value < 0 || !com.IsOnCurve() {
		return false
	}
	p := twistededwards.PointAffine{}
	i := big.NewInt(int64(value))
	p.ScalarMul(&curveParams.Base, i)
	temp := twistededwards.PointAffine{}
	temp.ScalarMul(&h, r)
	p.Add(&p, &temp)
	return p.Equal(com)
}

func ProveCommit(value int, r *big.Int, comBytes,  m []byte) (*twistededwards.PointAffine, *big.Int, *big.Int, error) {
	// commitment
	t := twistededwards.PointAffine{}
	i := big.NewInt(int64(value))
	r1, err := Random()
	if err != nil {
		return nil, nil, nil, err
	}
	r2, err := Random()
	if err != nil {
		return nil, nil, nil, err
	}
	t.ScalarMul(&curveParams.Base, r1)
	temp := twistededwards.PointAffine{}
	temp.ScalarMul(&h, r2)
	t.Add(&t, &temp)
	// challenge
	c := new(big.Int)
	c.SetBytes(hashTranscript(&t, comBytes, m))
	// response
	s1 := new(big.Int)
	s2 := new(big.Int)
	s1.Mul(i, c)
	s1.Add(s1, r1)
	s1.Mod(s1, &order)
	s2.Mul(r, c)
	s2.Add(s2, r2)
	s2.Mod(s2, &order)
	return &t, s1, s2, nil
}

func CommitProofToBytes(t *twistededwards.PointAffine, s1, s2 *big.Int) []byte {
	proofBytes := make([]byte, 96)
	tBytes := t.Marshal()
	copy(proofBytes[:32], tBytes)
	s1.FillBytes(proofBytes[32:64])
	s2.FillBytes(proofBytes[64:])
	return proofBytes
}

func CheckCommitProofBytes(proofBytes, comBytes, m []byte) bool {
	if len(proofBytes) != 96 {
		return false
	}
	tBytes := proofBytes[:32]
	// transform comBytes into a point and check it is on the curve
	t := twistededwards.PointAffine{}
	err := t.Unmarshal(tBytes)
	if err != nil || !t.IsOnCurve() {
		return false
	}
	s1Bytes := proofBytes[32:64]
	s1 := new(big.Int)
	s1.SetBytes(s1Bytes)
	s1.Mod(s1, &order)
	s2Bytes := proofBytes[64:]
	s2 := new(big.Int)
	s2.SetBytes(s2Bytes)
	s2.Mod(s2, &order)
	return CheckCommitProof(&t, s1, s2, comBytes, m)
}

func CheckCommitProof(t *twistededwards.PointAffine, s1, s2 *big.Int, comBytes, m []byte) bool {
	// transform comBytes into a point and check it is on the curve
	com := twistededwards.PointAffine{}
	err := com.Unmarshal(comBytes)
	if err != nil || !com.IsOnCurve() {
		return false
	}
	c := new(big.Int)
	c.SetBytes(hashTranscript(t, comBytes, m))

	left := twistededwards.PointAffine{}
	left.ScalarMul(&com, c)
	left.Add(&left, t)

	right := twistededwards.PointAffine{}
	temp := twistededwards.PointAffine{}
	right.ScalarMul(&curveParams.Base, s1)
	temp.ScalarMul(&h, s2)
	right.Add(&right, &temp)

	return left.Equal(&right)
}

func hashTranscript(t *twistededwards.PointAffine, comBytes, m []byte) []byte {
	h := sha256.New()
	h.Write(t.Marshal())
	h.Write(comBytes)
	h.Write(m)
	return h.Sum(nil)
}

func Random() (*big.Int, error) {
	for {
		k, err := rand.Int(rand.Reader, &order)
		if err != nil {
			return nil, err
		}

		if k.Sign() > 0 {
			return k, nil
		}
	}
}