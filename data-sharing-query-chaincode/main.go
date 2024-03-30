/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// QuerySmartContract provides functions for managing an Query
type QuerySmartContract struct {
	contractapi.Contract
}

// Query describes basic details of what makes up a simple query
// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically
type Query struct {
	Certificate    string `json:"Certificate"`
	DataDigest     string `json:"DataDigest"`
	DataRows       int    `json:"DatatRows"`
	InitiatorID    string `json:"InitiatorID"`
	InitiatorMSPID string `json:"InitiatorMSPID"`
	Legitimacy     string `json:"Legitimacy"`
	QueriedTable   string `json:"QueriedTable"`
	QueryDigest    string `json:"QueryDigest"`
	QueryID        string `json:"QueryID"`
	ServiceID      string `json:"ServiceID"`
	Timestamp      int    `json:"Timestamp"`
}

// InitLedger adds a base set of querys to the ledger
func (s *QuerySmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	return nil
}

// CreateQuery issues a new query to the world state with given details.
func (s *QuerySmartContract) CreateQuery(ctx contractapi.TransactionContextInterface, certificate, dataDigest string, dataRows int, initiatorID, initiatorMSPID, legitimacy, queriedTable, queryDigest, queryID, serviceID string, timestamp int) error {
	exists, err := s.QueryExists(ctx, queryID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the query %s already exists", queryID)
	}

	query := Query{
		Certificate:    certificate,
		DataDigest:     dataDigest,
		DataRows:       dataRows,
		InitiatorID:    initiatorID,
		InitiatorMSPID: initiatorMSPID,
		Legitimacy:     legitimacy,
		QueriedTable:   queriedTable,
		QueryDigest:    queryDigest,
		QueryID:        queryID,
		ServiceID:      serviceID,
		Timestamp:      timestamp,
	}
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return err
	}

	err = ctx.GetStub().SetEvent("Query", queryBytes)
	if err != nil {
		return fmt.Errorf("failed to SetEvent CreateQuery: %v", err)
	}

	return ctx.GetStub().PutState(queryID, queryBytes)
}

// ReadQuery returns the query stored in the world state with given id.
func (s *QuerySmartContract) ReadQuery(ctx contractapi.TransactionContextInterface, queryID string) (*Query, error) {
	queryBytes, err := ctx.GetStub().GetState(queryID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if queryBytes == nil {
		return nil, fmt.Errorf("the query %s does not exist", queryID)
	}

	var query Query
	err = json.Unmarshal(queryBytes, &query)
	if err != nil {
		return nil, err
	}

	return &query, nil
}

// QueryExists returns true when query with given ID exists in world state
func (s *QuerySmartContract) QueryExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	queryBytes, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return queryBytes != nil, nil
}

// GetAllQuerys returns all querys found in world state
func (s *QuerySmartContract) GetAllQuerys(ctx contractapi.TransactionContextInterface) ([]*Query, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all querys in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var querys []*Query
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var query Query
		err = json.Unmarshal(queryResponse.Value, &query)
		if err != nil {
			return nil, err
		}
		querys = append(querys, &query)
	}

	return querys, nil
}

func main() {
	queryChaincode, err := contractapi.NewChaincode(&QuerySmartContract{})
	if err != nil {
		log.Panicf("Error creating data-service-querying chaincode: %v", err)
	}

	if err := queryChaincode.Start(); err != nil {
		log.Panicf("Error starting data-service-querying chaincode: %v", err)
	}
}
