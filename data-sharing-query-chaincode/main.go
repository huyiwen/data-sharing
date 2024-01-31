package main

import (
        "log"
        "encoding/json"

        "github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
        contractapi.Contract
}

// Asset describes basic details of what makes up a simple asset
// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically
type Query struct {
        DataDigest     string `json:"DataDigest"`
        DataRows       int    `json:"DatatRows"`
        InitiatorID    string `json:"InitiatorID"`
        InitiatorMSPID string `json:"InitiatorMSPID"`
        Legitimacy     bool   `json:"Legitimacy"`
        QueriedTable   string `json:"QueriedTable"`
        QueryDigest    string `json:"QueryDigest"`
        QueryID        string `json:"QueryID"`
        ServiceID      string `json:"ServiceID"`
        Timestamp      int    `json:"Timestamp"`
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
        return nil
}

// CreateAsset issues a new asset to the world state with given details.
func (s *SmartContract) PutQuery(ctx contractapi.TransactionContextInterface, id string, dataDigest string, dataRows int, initiatorID string, initiatorMSPID string, legitimacy bool, queriedTable string, queryStatement string, serviceID string, timestamp int) error {
        query := Query{
                DataDigest: dataDigest,
                DataRows: dataRows,
                InitiatorID: initiatorID,
                InitiatorMSPID: initiatorMSPID,
                Legitimacy: legitimacy,
                QueriedTable: queriedTable,
                QueryStatement: queryStatement,
                QueryID: id,
                ServiceID: serviceID,
                Timestamp: timestamp,
        }
        queryJSON, err := json.Marshal(query)
        if err != nil {
                return err
        }

        return ctx.GetStub().PutState(id, queryJSON)
}

// GetAllAssets returns all assets found in world state
func (s *SmartContract) GetAllQueries(ctx contractapi.TransactionContextInterface) ([]*Query, error) {
        // range query with empty string for startKey and endKey does an
        // open-ended query of all assets in the chaincode namespace.
        resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
        if err != nil {
                return nil, err
        }
        defer resultsIterator.Close()

        var queries []*Query
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
                queries = append(queries, &query)
        }

        return queries, nil
}

func main() {
        assetChaincode, err := contractapi.NewChaincode(&SmartContract{})
        if err != nil {
                log.Panicf("Error creating data-transfer chaincode: %v", err)
        }

        if err := assetChaincode.Start(); err != nil {
                log.Panicf("Error starting data-transfer chaincode: %v", err)
        }
}
