package main

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// 需要定义一个链码总体的结构体，这个结构体定义了关于资产调用的常用接口
type SmartContract struct {
	contractapi.Contract
}

// Service 表示服务的数据结构
type Service struct {
	ServiceName      string   `json:"serviceName"`
	ServiceID        string   `json:"serviceID"`
	Headers          []string `json:"headers"`
	SellerURL        string   `json:"sellerURL"`
	SellerPublicKey  string   `json:"sellerPublicKey"`
	Comment          string   `json:"comment"`
}
 
// Application 表示应用程序的数据结构
type Application struct {
	BuyerName       string   `json:"buyerName"`
	BuyerPublicKey  string   `json:"buyerPublicKey"`
	Headers         []string `json:"headers"`
}

// Token 表示令牌的数据结构
type Token struct {
	ApplierID  string   `json:"applierID"`
	Headers    []string `json:"headers"`
}

// TransferRecord 表示交易记录的数据结构
type TransferRecord struct {
	Commands   []string `json:"commands"`
	ApplierID  string   `json:"applierID"`
}

// CreateService 由seller调用，创建一个新的服务
func (s *SmartContract) CreateService(ctx contractapi.TransactionContextInterface, serviceName string, serviceID string, headers []string, sellerURL string, sellerPublicKey string, comment string) bool {
	service := Service{
		ServiceName:     serviceName,
		ServiceID:       serviceID,
		Headers:         headers,
		SellerURL:       sellerURL,
		SellerPublicKey: sellerPublicKey,
		Comment:         comment,
	}

	serviceAsBytes, err := json.Marshal(service)
	if err != nil {
		fmt.Println("Error marshaling service data:", err.Error())
		return false
	}

	key := fmt.Sprintf("Service%d", len(headers)) // 这里可以根据实际情况自定义key的生成规则
	err = ctx.GetStub().PutState(key, serviceAsBytes)
	if err != nil {
		fmt.Println("Error putting state:", err.Error())
		return false
	}

	return true
}

// GetServices 由buyer调用，获取所有的services
func (s *SmartContract) GetServices(ctx contractapi.TransactionContextInterface, startKey string, endKey string) ([]*Service, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var services []*Service

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var service Service
		err = json.Unmarshal(queryResponse.Value, &service)
		if err != nil {
			return nil, err
		}

		services = append(services, &service)
	}

	return services, nil
}

// FetchData 由buyer调用，记录一笔交易数据
func (s *SmartContract) FetchData(ctx contractapi.TransactionContextInterface, applierID string, usageAsBytes []byte) bool {
	// 这里使用GetTxID()作为唯一标识，你也可以使用其他方式生成唯一标识
	txID := ctx.GetStub().GetTxID()

	err := ctx.GetStub().PutState(txID, usageAsBytes)
	if err != nil {
		fmt.Println("Error putting state:", err.Error())
		return false
	}

	return true
}

func main() {
 
	chaincode, err := contractapi.NewChaincode(new(SmartContract))
	if err != nil {
		fmt.Printf("Error create chaincode: %s", err.Error())
		return
	}
 
	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting chaincode: %s", err.Error())
	}
}
