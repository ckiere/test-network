package main

import (
	"fmt"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/twistededwards"
	"log"
	"os"
)

const MaxBids = 10

type AuctionCircuit struct {
	// struct tags on a variable is optional
	// default uses variable name and secret visibility.
	Values [MaxBids]frontend.Variable
	Rs [MaxBids]frontend.Variable
	ComsX [MaxBids]frontend.Variable `gnark:",public"`
	ComsY [MaxBids]frontend.Variable `gnark:",public"`
	WinningValue frontend.Variable
	WinningR frontend.Variable
	WinningComX frontend.Variable `gnark:",public"`
	WinningComY frontend.Variable `gnark:",public"`
}

// Define declares the circuit constraints
func (circuit *AuctionCircuit) Define(curveID ecc.ID, cs *frontend.ConstraintSystem) error {
	// constants
	curve, _ := twistededwards.NewEdCurve(ecc.BLS12_381)
	// check winning commitment
	circuit.CheckCommitment(curve, circuit.WinningValue, circuit.WinningR, circuit.WinningComX, circuit.WinningComY, cs)
	// check all other bids (valid commitment and value lower than winning bid)
	for i := 0; i < MaxBids; i++ {
		circuit.CheckCommitment(curve, circuit.Values[i], circuit.Rs[i], circuit.ComsX[i], circuit.ComsY[i], cs)
		cs.AssertIsLessOrEqual(circuit.Values[i], circuit.WinningValue)
	}
	return nil
}

func (circuit *AuctionCircuit) CheckCommitment(curve twistededwards.EdCurve, value, r, comX, comY frontend.Variable, cs *frontend.ConstraintSystem) {
	// com = g^value h^r
	com := twistededwards.Point{}
	com.ScalarMulFixedBase(cs, curve.BaseX, curve.BaseY, value, curve)
	temp := twistededwards.Point{}
	temp.ScalarMulFixedBase(cs, hx, hy, r, curve)
	com.AddGeneric(cs, &com, &temp, curve)
	cs.AssertIsEqual(com.X, comX)
	cs.AssertIsEqual(com.Y, comY)
}

func main() {
	var circuit AuctionCircuit

	// compiles our circuit into a R1CS
	r1cs, err := frontend.Compile(ecc.BLS12_381, backend.GROTH16, &circuit)
	if err != nil {
		log.Fatalf("compilation of the circuit failed: %v", err)
	}
	fmt.Printf("Nb constraints: %v", r1cs.GetNbConstraints())
	pk, vk, err := groth16.Setup(r1cs)
	if err != nil {
		log.Fatalf("setup failed: %v", err)
	}
	file, err := os.OpenFile("circuit", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = r1cs.WriteTo(file)
	if err != nil {
		panic(err)
	}
	pkFile, err := os.OpenFile("pk", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if err != nil {
		panic(err)
	}
	defer pkFile.Close()
	_, err = pk.WriteTo(pkFile)
	if err != nil {
		panic(err)
	}
	vkFile, err := os.OpenFile("vk", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if err != nil {
		panic(err)
	}
	defer vkFile.Close()
	_, err = vk.WriteTo(vkFile)
	if err != nil {
		panic(err)
	}

	testProof(r1cs, pk, vk)

	/*com, r, _ := Commit(100)
	fmt.Println(CheckCommit(100, r, com))
	c, s1, s2, err := ProveCommit(100, r, com.Marshal(), nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(CheckCommitProof(c, s1, s2, com.Marshal(), nil))*/

}

func testProof(r1cs frontend.CompiledConstraintSystem, pk groth16.ProvingKey, vk groth16.VerifyingKey) {
	fmt.Println("Building proof")
	var witness AuctionCircuit
	var solution AuctionCircuit
	n := 10

	winningValue := 500

	witness.WinningValue.Assign(winningValue)
	winningCom, winningR, _ := Commit(winningValue)
	witness.WinningR.Assign(winningR)
	witness.WinningComX.Assign(winningCom.X)
	witness.WinningComY.Assign(winningCom.Y)
	solution.WinningComX.Assign(winningCom.X)
	solution.WinningComY.Assign(winningCom.Y)

	for i := 0; i < n ; i++ {
		value := 100 + i
		witness.Values[i].Assign(value)
		com, r, _ := Commit(value)
		witness.Rs[i].Assign(r)
		witness.ComsX[i].Assign(com.X)
		witness.ComsY[i].Assign(com.Y)
		solution.ComsX[i].Assign(com.X)
		solution.ComsY[i].Assign(com.Y)
	}
	// fill non used bids with a value of 0
	for i := n; i < MaxBids ; i++ {
		value := 0
		com, r, _ := Commit(value)
		witness.Values[i].Assign(value)
		witness.Rs[i].Assign(r)

		witness.ComsX[i].Assign(com.X)
		witness.ComsY[i].Assign(com.Y)
		solution.ComsX[i].Assign(com.X)
		solution.ComsY[i].Assign(com.Y)
	}
	proof, err := groth16.Prove(r1cs, pk, &witness)
	if err != nil {
		log.Fatalf("prove failed: %v", err)
	}
	fmt.Println("Verifying")
	err = groth16.Verify(proof, vk, &solution)
	if err != nil {
		log.Fatalf("verify failed :%v", err)
	}
}