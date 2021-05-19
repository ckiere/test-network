package main

import (
	"bytes"
	"client-auctioneer/crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc"
	mimc2 "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"golang.org/x/crypto/nacl/box"
	"os"
)

const channelName = "auction"
const SellerPkSize = 32
const MaxBids = 50

// Auction data
type Auction struct {
	Type         string                    `json:"objectType"`
	ItemSold     string                    `json:"item"`
	Seller       string                    `json:"seller"`
	SellerPk	 [SellerPkSize]byte        `json:"sellerPk"`
	Commitments  map[string] string        `json:"commitments"`
	EncryptedBids map[string] EncryptedBid `json:"encryptedBids"`
	WinningBid   string                    `json:"winningBid"`
	Proof        []byte
	Status       string                    `json:"status"`
}

// EncryptedBid contains the values needed to open a commitment to a bid, encrypted with the public key of the seller
type EncryptedBid struct {
	Type     string `json:"objectType"`
	Data     []byte    `json:"data"`
	Bidder   string `json:"bidder"`
}

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
	argc := len(os.Args)
	if argc > 1 {
		cmd := os.Args[1]
		if cmd == "client" {
			if argc == 5 {
				launchClient(os.Args[2], os.Args[3], os.Args[4])
			} else {
				fmt.Println("Wrong number of arguments")
			}
		} else {
			fmt.Println("Unknown command")
		}
	} else {
		fmt.Println("No command given as argument")
	}
}

func launchClient(username string, auctionID, itemName string) {
	sdk, err := fabsdk.New(config.FromFile("connection-org1.yaml"))
	if err != nil {
		panic(err)
	}

	dacClientChannelContext := sdk.ChannelContext(channelName, fabsdk.WithUser(username), fabsdk.WithOrg("org1"))
	client, err := channel.New(dacClientChannelContext)
	if err != nil {
		panic(err)
	}

	// create the auctioneer public key
	pk, sk, err := box.GenerateKey(rand.Reader)
	/*if err != nil {
		panic(err)
	}
	pkBase64 := base64.StdEncoding.EncodeToString(pk[:])
	// start auction
	client.Execute(channel.Request{ChaincodeID: "blindauction", Fcn: "CreateAuction", Args: [][]byte{[]byte(auctionID),
		[]byte(itemName), []byte(pkBase64)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com"))
	// pause to wait for the second phase of the auction
	time.Sleep(time.Duration(30) * time.Second)

	// close auction
	client.Execute(channel.Request{ChaincodeID: "blindauction", Fcn: "CloseAuction", Args: [][]byte{[]byte(auctionID)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com"))
	// pause to wait for the second phase of the auction
	time.Sleep(time.Duration(30) * time.Second)

	// end auction
	client.Execute(channel.Request{ChaincodeID: "blindauction", Fcn: "EndAuction", Args: [][]byte{[]byte(auctionID)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com"))
*/
	// query the auction
	response, _ := client.Query(channel.Request{ChaincodeID: "blindauction", Fcn: "QueryAuction", Args: [][]byte{[]byte(auctionID)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com"))
	var auction Auction
	auctionBytes := response.Payload
	fmt.Print(auctionBytes)
	err = json.Unmarshal(auctionBytes, &auction)
	if err != nil {
		panic(err)
	}

	// get the encrypted bids
	var witness AuctionCircuit
	encryptedBids := auction.EncryptedBids
	commitments := auction.Commitments
	bestPrice := 0
	bestID := ""
	n := 0
	var bestCom, bestR []byte
	for name, encryptedBid := range encryptedBids {
		price, r, err := crypto.Decrypt(encryptedBid.Data, pk, sk)
		comBase64 := commitments[name]
		comBytes, err := base64.StdEncoding.DecodeString(comBase64)
		if err != nil || !crypto.CheckCommit(comBytes, price, r) {
			// TODO
			fmt.Println("err")
		}

		witness.Values[n].Assign(price)
		witness.Rs[n].Assign(r)
		witness.Coms[n].Assign(comBytes)

		if price > bestPrice {
			bestPrice = price
			bestID = name
			bestCom = comBytes
			bestR = r
		}
		n++
	}

	// Compute proof
	witness.WinningValue.Assign(bestPrice)
	witness.WinningR.Assign(bestR)
	witness.WinningCom.Assign(bestCom)

	// fill non used bids with a value of 0
	for i := n; i < MaxBids ; i++ {
		value := 0
		r := []byte{0, 0, 0, 0}
		witness.Values[i].Assign(value)
		witness.Rs[i].Assign(r)
		com := Commit(value, r)
		witness.Coms[i].Assign(com)
	}

	// load circuit
	circuitFile, err := os.Open("circuit")
	defer circuitFile.Close()
	r1cs := groth16.NewCS(ecc.BN254)
	_, err = r1cs.ReadFrom(circuitFile)
	if err != nil {
		panic(err)
	}

	// load proving key
	prkFile, err := os.Open("pk")
	defer prkFile.Close()
	prk := groth16.NewProvingKey(ecc.BN254)
	_, err = prk.ReadFrom(prkFile)
	if err != nil {
		panic(err)
	}

	proof, err := groth16.Prove(r1cs, prk, &witness)
	if err != nil {
		panic(err)
	}

	var proofBuf bytes.Buffer
	proof.WriteTo(&proofBuf)

	// declare winner
	client.Execute(channel.Request{ChaincodeID: "blindauction", Fcn: "DeclareWinner", Args: [][]byte{[]byte(auctionID),
		[]byte(bestID), []byte(base64.StdEncoding.EncodeToString(proofBuf.Bytes()))}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com"))

}

func Commit(value int, r []byte) []byte {
	valueBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valueBytes, uint32(value))
	mc := mimc2.NewMiMC("seed")
	mc.Write(r)
	mc.Write(valueBytes)
	return mc.Sum(nil)
}
