package main

import (
	"bytes"
	"client-auctioneer/crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	twistededwards2 "github.com/consensys/gnark-crypto/ecc/bls12-381/twistededwards"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/twistededwards"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"golang.org/x/crypto/nacl/box"
	"math/big"
	"os"
	"time"
)

const channelName = "auction"
const chaincodeID = "blindauction"
const SellerPkSize = 32
const MaxBids = 10

var hx = new(fr.Element).SetString("51295569138718539371092613972351202357326289069440880621285444911501458459494")
var hy = new(fr.Element).SetString("49831129265363587078046764490824666482509464638593900758877649985443393819454")

// Auction data
type Auction struct {
	Type         string                    `json:"objectType"`
	ItemSold     string                    `json:"item"`
	Seller       string                    `json:"seller"`
	SellerPk	 [SellerPkSize]byte        `json:"sellerPk"`
	Commitments  map[string] []byte        `json:"commitments"`
	EncryptedBids map[string] EncryptedBid `json:"encryptedBids"`
	InvalidSet   string                    `json:"invalidSet"`
	WinningBid   string                    `json:"winningBid"`
	Proof        []byte                    `json:"proof"`
	Status       string                    `json:"status"`
}

type Bid struct {
	Type     string `json:"objectType"`
	Price    int    `json:"price"`
	R        big.Int `json:"r"`
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
	argc := len(os.Args)
	if argc > 1 {
		cmd := os.Args[1]
		if cmd == "client" {
			if argc > 5 {
				launchClient(os.Args[2], os.Args[3], os.Args[4], os.Args[5:])
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

func launchClient(username string, auctionID, itemName string, endpoints []string) {
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
	if err != nil {
		panic(err)
	}
	pkBase64 := base64.StdEncoding.EncodeToString(pk[:])
	// start auction
	client.Execute(channel.Request{ChaincodeID: chaincodeID, Fcn: "CreateAuction", Args: [][]byte{[]byte(auctionID),
		[]byte(itemName), []byte(pkBase64)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))
	// pause to wait for the second phase of the auction
	time.Sleep(time.Duration(30) * time.Second)

	// close auction
	client.Execute(channel.Request{ChaincodeID: chaincodeID, Fcn: "CloseAuction", Args: [][]byte{[]byte(auctionID)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))
	// pause to wait for the second phase of the auction
	time.Sleep(time.Duration(30) * time.Second)

	// end auction
	client.Execute(channel.Request{ChaincodeID: chaincodeID, Fcn: "EndAuction", Args: [][]byte{[]byte(auctionID)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))

	// query the auction
	response, _ := client.Query(channel.Request{ChaincodeID: chaincodeID, Fcn: "QueryAuction", Args: [][]byte{[]byte(auctionID)}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))
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
	invalidBids := make(map[string] Bid)
	bestPrice := -1
	bestID := ""
	n := 0
	var bestCom twistededwards2.PointAffine
	var bestR *big.Int
	// check the number of bids is not greater than the max number of bids in a proof
	// if it is, the auction requires manual intervention
	if len(encryptedBids) > MaxBids {
		panic("Too many bids in the auction")
	}
	for name, encryptedBid := range encryptedBids {
		comBytes, exists := commitments[name]
		// only take the bid into account if there was a commitment for it
		// this should always be true, otherwise there is a flaw in the smart contract
		if exists {
			price, r, err := crypto.Decrypt(encryptedBid.Data, pk, sk)
			com := twistededwards2.PointAffine{}
			err2 := com.Unmarshal(comBytes)
			// check the decryption is valid
			if err == nil && err2 == nil && crypto.CheckCommit(price, r, &com) {
				witness.Values[n].Assign(price)
				witness.Rs[n].Assign(r)
				witness.ComsX[n].Assign(com.X)
				witness.ComsY[n].Assign(com.Y)

				if price > bestPrice {
					bestPrice = price
					bestID = name
					bestCom = com
					bestR = r
				}
				n++
			} else {
				fmt.Errorf("decryption of bid %v invalid", name)
				invalidBids[name] = Bid{
					Type:  "bid",
					Price: price,
					R:     *r,
				}
			}
		}
	}

	// Compute proof
	var proofBytes []byte
	if n > 0 {
		witness.WinningValue.Assign(bestPrice)
		witness.WinningR.Assign(bestR)
		witness.WinningComX.Assign(bestCom.X)
		witness.WinningComY.Assign(bestCom.Y)

		// fill non used bids with a value of 0
		for i := n; i < MaxBids; i++ {
			value := 0
			com, r, _ := crypto.Commit(value)
			witness.Values[i].Assign(value)
			witness.Rs[i].Assign(r)
			witness.ComsX[i].Assign(com.X)
			witness.ComsY[i].Assign(com.Y)
		}

		// load circuit
		circuitFile, err := os.Open("circuit")
		defer circuitFile.Close()
		r1cs := groth16.NewCS(ecc.BLS12_381)
		_, err = r1cs.ReadFrom(circuitFile)
		if err != nil {
			panic(err)
		}

		// load proving key
		prkFile, err := os.Open("pk")
		defer prkFile.Close()
		prk := groth16.NewProvingKey(ecc.BLS12_381)
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
		proofBytes = proofBuf.Bytes()
	}
	// put invalid bids into a JSON
	invalidSet, err := json.Marshal(invalidBids)
	if err != nil {
		panic(err)
	}

	// declare winner
	client.Execute(channel.Request{ChaincodeID: chaincodeID, Fcn: "DeclareWinner", Args: [][]byte{[]byte(auctionID),
		[]byte(bestID), []byte(base64.StdEncoding.EncodeToString(proofBytes)), invalidSet}},
		channel.WithRetry(retry.DefaultChannelOpts), channel.WithTargetEndpoints(endpoints...))

}
