/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-samples/auction/chaincode-go/crypto"
)

type SmartContract struct {
	contractapi.Contract
}

const SellerPkSize = 32

// Auction data
type Auction struct {
	Type         string                    `json:"objectType"`
	ItemSold     string                    `json:"item"`
	Seller       string                    `json:"seller"`
	SellerPk	 [SellerPkSize]byte        `json:"sellerPk"`
	Commitments  map[string] []byte        `json:"commitments"`
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

// CreateAuction creates on auction on the public channel. The identity that
// submits the transaction becomes the seller of the auction
func (s *SmartContract) CreateAuction(ctx contractapi.TransactionContextInterface, auctionID, itemsold, sellerPk string) error {

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	// get org of submitting client
	clientOrgID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	// get seller public key
	pkBytes, err := base64.StdEncoding.DecodeString(sellerPk)
	if err != nil || len(pkBytes) != SellerPkSize {
		return fmt.Errorf("invalid seller public key")
	}
	var sellerPkBytes [SellerPkSize]byte
	copy(sellerPkBytes[:], pkBytes)

	// Create auction
	coms := make(map[string] []byte)
	revealedBids := make(map[string]EncryptedBid)

	auction := Auction{
		Type:         "auction",
		ItemSold:     itemsold,
		Seller:       clientID,
		SellerPk:     sellerPkBytes,
		Commitments:  coms,
		EncryptedBids: revealedBids,
		WinningBid:   "",
		Status:       "open",
	}

	auctionBytes, err := json.Marshal(auction)
	if err != nil {
		return err
	}

	// put auction into state
	err = ctx.GetStub().PutState(auctionID, auctionBytes)
	if err != nil {
		return fmt.Errorf("failed to put auction in public data: %v", err)
	}

	// set the seller of the auction as an endorser
	err = setAssetStateBasedEndorsement(ctx, auctionID, clientOrgID)
	if err != nil {
		return fmt.Errorf("failed setting state based endorsement for new organization: %v", err)
	}

	return nil
}

// SendCommitment is used by the anonymous bidders to submit a commitment to a bid
func (s *SmartContract) SendCommitment(ctx contractapi.TransactionContextInterface, auctionID, commitment, proof string) (string, error) {
	// verify the proof of knowledge of opening values
	comBytes, err := base64.StdEncoding.DecodeString(commitment)
	if err != nil {
		return "", err
	}
	proofBytes, err := base64.StdEncoding.DecodeString(proof)
	if err != nil {
		return "", err
	}
	if !crypto.CheckCommitProofBytes(proofBytes, comBytes, nil) {
		return "", fmt.Errorf("invalid proof")
	}
	// get the auction from state
	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	var auctionJSON Auction

	if auctionBytes == nil {
		return "", fmt.Errorf("auction not found")
	}
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return "", fmt.Errorf("failed to create auction object JSON")
	}

	// the auction needs to be open for users to add their bid
	Status := auctionJSON.Status
	if Status != "open" {
		return "", fmt.Errorf("cannot join closed or ended auction")
	}

	// use the transaction ID as a key for the commitment
	txID := ctx.GetStub().GetTxID()
	auctionJSON.Commitments[txID] = comBytes

	newAuctionBytes, _ := json.Marshal(auctionJSON)
	err = ctx.GetStub().PutState(auctionID, newAuctionBytes)
	if err != nil {
		return "", fmt.Errorf("failed to update auction")
	}
	return txID, nil
}

// RevealBid is used by a bidder to reveal their bid after the auction is closed
func (s *SmartContract) RevealBid(ctx contractapi.TransactionContextInterface, auctionID, txID, bidder, data, proof string) error {
	dataBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return fmt.Errorf("invalid data")
	}

	// get auction from public state
	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction %v: %v", auctionID, err)
	}
	if auctionBytes == nil {
		return fmt.Errorf("Auction interest object %v not found", auctionID)
	}

	var auctionJSON Auction
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// check that the auction is closed
	Status := auctionJSON.Status
	if Status != "closed" {
		return fmt.Errorf("cannot reveal bid for open or ended auction")
	}

	// check the commitment exists in the state
	comBytes, exists := auctionJSON.Commitments[txID]
	if !exists {
		return fmt.Errorf("commitment does not exist")
	}

	// check the proof of knowledge of opening values
	proofBytes, err := base64.StdEncoding.DecodeString(proof)
	if err != nil {
		return err
	}
	if !crypto.CheckCommitProofBytes(proofBytes, comBytes, dataBytes) {
		return fmt.Errorf("invalid proof")
	}
	// add the new revealed bid to the list
	NewBid := EncryptedBid{
		Type:     "bid",
		Data:    dataBytes,
		Bidder:   bidder,
	}
	encryptedBids := make(map[string]EncryptedBid)
	encryptedBids = auctionJSON.EncryptedBids
	encryptedBids[txID] = NewBid
	auctionJSON.EncryptedBids = encryptedBids

	newAuctionBytes, _ := json.Marshal(auctionJSON)

	// put auction with bid added back into state
	err = ctx.GetStub().PutState(auctionID, newAuctionBytes)
	if err != nil {
		return fmt.Errorf("failed to update auction: %v", err)
	}

	return nil
}

// CloseAuction can be used by the seller to close the auction. This prevents
// bids from being added to the auction, and allows users to reveal their bid
func (s *SmartContract) CloseAuction(ctx contractapi.TransactionContextInterface, auctionID string) error {

	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction %v: %v", auctionID, err)
	}

	if auctionBytes == nil {
		return fmt.Errorf("Auction interest object %v not found", auctionID)
	}

	var auctionJSON Auction
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// the auction can only be closed by the seller

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	Seller := auctionJSON.Seller
	if Seller != clientID {
		return fmt.Errorf("auction can only be closed by seller: %v", err)
	}

	Status := auctionJSON.Status
	if Status != "open" {
		return fmt.Errorf("cannot close auction that is not open")
	}

	auctionJSON.Status = "closed"

	closedAuction, _ := json.Marshal(auctionJSON)

	err = ctx.GetStub().PutState(auctionID, closedAuction)
	if err != nil {
		return fmt.Errorf("failed to close auction: %v", err)
	}

	return nil
}

// EndAuction changes the status to ended
func (s *SmartContract) EndAuction(ctx contractapi.TransactionContextInterface, auctionID string) error {

	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction %v: %v", auctionID, err)
	}

	if auctionBytes == nil {
		return fmt.Errorf("Auction interest object %v not found", auctionID)
	}

	var auctionJSON Auction
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// Check that the auction is being ended by the seller

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	Seller := auctionJSON.Seller
	if Seller != clientID {
		return fmt.Errorf("auction can only be ended by seller: %v", err)
	}

	Status := auctionJSON.Status
	if Status != "closed" {
		return fmt.Errorf("Can only end a closed auction")
	}

	// get the list of revealed bids
	if len(auctionJSON.EncryptedBids) == 0 {
		return fmt.Errorf("No bids have been revealed, cannot end auction: %v", err)
	}
	auctionJSON.Status = "ended"

	endedAuction, _ := json.Marshal(auctionJSON)

	err = ctx.GetStub().PutState(auctionID, endedAuction)
	if err != nil {
		return fmt.Errorf("failed to end auction: %v", err)
	}
	return nil
}

// DeclareWinner sets the winner of an auction
func (s *SmartContract) DeclareWinner(ctx contractapi.TransactionContextInterface, auctionID, winningBidId, proof string) error {
	auctionBytes, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return fmt.Errorf("failed to get auction %v: %v", auctionID, err)
	}

	if auctionBytes == nil {
		return fmt.Errorf("Auction interest object %v not found", auctionID)
	}

	var auctionJSON Auction
	err = json.Unmarshal(auctionBytes, &auctionJSON)
	if err != nil {
		return fmt.Errorf("failed to create auction object JSON: %v", err)
	}

	// Check that the auction is being ended by the seller

	// get ID of submitting client
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client identity %v", err)
	}

	Seller := auctionJSON.Seller
	if Seller != clientID {
		return fmt.Errorf("auction can only be ended by seller: %v", err)
	}

	Status := auctionJSON.Status
	if Status != "ended" {
		return fmt.Errorf("can only declare the winner of an ended auction")
	}
	// Set the winner
	proofBytes, err := base64.StdEncoding.DecodeString(proof)
	if err != nil {
		return fmt.Errorf("invalid proof format")
	}
	auctionJSON.Proof = proofBytes
	auctionJSON.WinningBid = winningBidId
	// Save auction
	endedAuction, _ := json.Marshal(auctionJSON)
	err = ctx.GetStub().PutState(auctionID, endedAuction)
	if err != nil {
		return fmt.Errorf("failed to set auction winner: %v", err)
	}
	return nil
}