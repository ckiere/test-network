/*
SPDX-License-Identifier: Apache-2.0
*/

package auction

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// QueryAuction allows all members of the channel to read a public auction
func (s *SmartContract) QueryAuction(ctx contractapi.TransactionContextInterface, auctionID string) (string, error) {

	auctionJSON, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return "", fmt.Errorf("failed to get auction object %v: %v", auctionID, err)
	}
	if auctionJSON == nil {
		return "", fmt.Errorf("auction does not exist")
	}

	return string(auctionJSON), nil
}

func (s *SmartContract) QueryAuctioneerPk(ctx contractapi.TransactionContextInterface, auctionID string) (string, error) {
	auctionJSON, err := ctx.GetStub().GetState(auctionID)
	if err != nil {
		return "", fmt.Errorf("failed to get auction object %v: %v", auctionID, err)
	}
	if auctionJSON == nil {
		return "", fmt.Errorf("auction does not exist")
	}

	var auction *Auction
	err = json.Unmarshal(auctionJSON, &auction)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(auction.SellerPk[:]), nil
}


// GetID is an internal helper function to allow users to get their identity
func (s *SmartContract) GetID(ctx contractapi.TransactionContextInterface) (string, error) {

	// Get the MSP ID of submitting client identity
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get verified MSPID: %v", err)
	}

	return clientID, nil
}