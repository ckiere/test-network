package main

import (
	"encoding/binary"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc"
	mimc2 "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
	"log"
	"os"
)

const MaxBids = 50

type AuctionCircuit struct {
	// struct tags on a variable is optional
	// default uses variable name and secret visibility.
	Values [MaxBids]frontend.Variable
	Rs [MaxBids]frontend.Variable
	Coms [MaxBids]frontend.Variable `gnark:",public"`
	WinningValue frontend.Variable
	WinningR frontend.Variable
	WinningCom frontend.Variable `gnark:",public"`
}

// Define declares the circuit constraints
func (circuit *AuctionCircuit) Define(curveID ecc.ID, cs *frontend.ConstraintSystem) error {
	// constants
	mc, _ := mimc.NewMiMC("seed", curveID)
	shift := cs.Constant(1 << 32)
	// check winning commitment
	circuit.CheckCommitment(mc, circuit.WinningValue, circuit.WinningR, circuit.WinningCom, shift, cs)
	// check all other bids (valid commitment and value lower than winning bid)
	for i := 0; i < MaxBids; i++ {
		circuit.CheckCommitment(mc, circuit.Values[i], circuit.Rs[i], circuit.Coms[i], shift, cs)
		cs.AssertIsLessOrEqual(circuit.Values[i], circuit.WinningValue)
	}
	return nil
}

func (circuit *AuctionCircuit) CheckCommitment(mc mimc.MiMC, value, r, com, shift frontend.Variable, cs *frontend.ConstraintSystem) {
	conc := cs.Add(cs.Mul(shift, r), value) // R || Value
	hash := mc.Hash(cs, conc)
	cs.AssertIsEqual(com, hash)
}

func main() {
	var circuit AuctionCircuit

	// compiles our circuit into a R1CS
	r1cs, err := frontend.Compile(ecc.BN254, backend.GROTH16, &circuit)
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
}

func testProof(r1cs frontend.CompiledConstraintSystem, pk groth16.ProvingKey, vk groth16.VerifyingKey) {
	fmt.Println("Building proof")
	var witness AuctionCircuit
	var solution AuctionCircuit
	n := 10

	winningValue := 500
	r := []byte{10, 20, 30, 40} // just to test
	witness.WinningValue.Assign(winningValue)
	witness.WinningR.Assign(r)
	winningCom := Commit(winningValue, r)
	witness.WinningCom.Assign(winningCom)
	solution.WinningCom.Assign(winningCom)

	for i := 0; i < n ; i++ {
		value := 100 + i
		witness.Values[i].Assign(value)
		witness.Rs[i].Assign(r)
		com := Commit(value, r)
		witness.Coms[i].Assign(com)
		solution.Coms[i].Assign(com)
	}
	// fill non used bids with a value of 0
	for i := n; i < MaxBids ; i++ {
		value := 0
		r := []byte{0, 0, 0, 0}
		witness.Values[i].Assign(value)
		witness.Rs[i].Assign(r)
		com := Commit(value, r)
		witness.Coms[i].Assign(com)
		solution.Coms[i].Assign(com)
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

func Commit(value int, r []byte) []byte {
	valueBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valueBytes, uint32(value))
	mc := mimc2.NewMiMC("seed")
	mc.Write(r)
	mc.Write(valueBytes)
	return mc.Sum(nil)
}