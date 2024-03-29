package main

import (
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func (c *TokenERC721Contract) BalanceOfByURI(ctx contractapi.TransactionContextInterface, owner string, tokenURI string) int {

	// Check if contract has been intilized first
	initialized, err := checkInitialized(ctx)
	if err != nil {
		panic("failed to check if contract is already initialized:" + err.Error())
	}
	if !initialized {
		panic("Contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	// There is a key record for every non-fungible token in the format of balancePrefix.owner.tokenId.
	// BalanceOf() queries for and counts all records matching balancePrefix.owner.*

	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey(balancePrefix, []string{owner})
	if err != nil {
		panic("Error creating asset chaincode:" + err.Error())
	}

	// Count the number of returned composite keys
	balance := 0
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return 0
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(response.Key)
		if err != nil {
			return 0
		}

		nft, err := _readNFT(ctx, compositeKeyParts[1])
		if err != nil {
			return 0
		}

		if nft.TokenURI == tokenURI {
			balance++
		}
	}
	return balance
}

func (c *TokenERC721Contract) BalanceOfByURIPrefix(ctx contractapi.TransactionContextInterface, owner string, tokenURIPrefix string) int {

	// Check if contract has been intilized first
	initialized, err := checkInitialized(ctx)
	if err != nil {
		panic("failed to check if contract is already initialized:" + err.Error())
	}
	if !initialized {
		panic("Contract options need to be set before calling any function, call Initialize() to initialize contract")
	}

	// There is a key record for every non-fungible token in the format of balancePrefix.owner.tokenId.
	// BalanceOf() queries for and counts all records matching balancePrefix.owner.*

	iterator, err := ctx.GetStub().GetStateByPartialCompositeKey(balancePrefix, []string{owner})
	if err != nil {
		panic("Error creating asset chaincode:" + err.Error())
	}

	// Count the number of returned composite keys
	balance := 0
	for iterator.HasNext() {
		response, err := iterator.Next()
		if err != nil {
			return 0
		}

		_, compositeKeyParts, err := ctx.GetStub().SplitCompositeKey(response.Key)
		if err != nil {
			return 0
		}

		nft, err := _readNFT(ctx, compositeKeyParts[1])
		if err != nil {
			return 0
		}

		if strings.HasPrefix(nft.TokenURI, tokenURIPrefix) {
			balance++
		}
	}
	return balance
}
