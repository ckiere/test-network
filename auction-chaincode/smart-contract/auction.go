/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-samples/auction/chaincode-go/commitment"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

// Auction data
type Auction struct {
	Type         string             `json:"objectType"`
	ItemSold     string             `json:"item"`
	Seller       string             `json:"seller"`
	Commitments  map[string] string `json:"commitments"`
	RevealedBids map[string]Bid     `json:"revealedBids"`
	WinningBid   string             `json:"winningBid"`
	Status       string             `json:"status"`
}

// Bid is the structure of a revealed bid
type Bid struct {
	Type     string `json:"objectType"`
	Price    int    `json:"price"`
	Bidder   string `json:"bidder"`
}

// CreateAuction creates on auction on the public channel. The identity that
// submits the transaction becomes the seller of the auction
func (s *SmartContract) CreateAuction(ctx contractapi.TransactionContextInterface, auctionID string, itemsold string) error {

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

	// Create auction
	coms := make(map[string] string)
	revealedBids := make(map[string]Bid)

	auction := Auction{
		Type:         "auction",
		ItemSold:     itemsold,
		Seller:       clientID,
		Commitments:  coms,
		RevealedBids: revealedBids,
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
func (s *SmartContract) SendCommitment(ctx contractapi.TransactionContextInterface, auctionID string, commitment string) (string, error) {

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
	auctionJSON.Commitments[txID] = commitment

	newAuctionBytes, _ := json.Marshal(auctionJSON)
	err = ctx.GetStub().PutState(auctionID, newAuctionBytes)
	if err != nil {
		return "", fmt.Errorf("failed to update auction")
	}
	return txID, nil
}

// RevealBid is used by a bidder to reveal their bid after the auction is closed
func (s *SmartContract) RevealBid(ctx contractapi.TransactionContextInterface, auctionID string, txID string, bidder string, price int, rand string) error {
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

	// check the offered price is valid
	if price <= 0 {
		return fmt.Errorf("invalid price")
	}

	// check the commitment from the state
	com, exists := auctionJSON.Commitments[txID]
	if !exists {
		return fmt.Errorf("commitment does not exist")
	}
	// decode base64 commitment and randomness
	comBytes, err := base64.StdEncoding.DecodeString(com)
	if err != nil {
		return fmt.Errorf("invalid commitment format")
	}
	randBytes, err := base64.StdEncoding.DecodeString(rand)
	if err != nil {
		return fmt.Errorf("invalid randomness format")
	}
	// check the commitment is valid
	if !commitment.CheckCommit(comBytes, price, randBytes) {
		return fmt.Errorf("invalid bid or rand")
	}

	// add the new revealed bid to the list
	NewBid := Bid{
		Type:     "bid",
		Price:    price,
		Bidder:   bidder,
	}
	revealedBids := make(map[string]Bid)
	revealedBids = auctionJSON.RevealedBids
	revealedBids[txID] = NewBid
	auctionJSON.RevealedBids = revealedBids

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

// EndAuction both changes the auction status to ended and calculates the winners
// of the auction
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
	revealedBidMap := auctionJSON.RevealedBids
	if len(auctionJSON.RevealedBids) == 0 {
		return fmt.Errorf("No bids have been revealed, cannot end auction: %v", err)
	}

	bestPrice := 0
	// determine the highest bid
	// TODO ZKP

	for id, bid := range revealedBidMap {
		if bid.Price > bestPrice {
			bestPrice = bid.Price
			auctionJSON.WinningBid = id
		}
	}

	auctionJSON.Status = "ended"

	closedAuction, _ := json.Marshal(auctionJSON)

	err = ctx.GetStub().PutState(auctionID, closedAuction)
	if err != nil {
		return fmt.Errorf("failed to close auction: %v", err)
	}
	return nil
}
