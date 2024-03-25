# 可信共享

- README.md: 本文件
- data-sharing-chaincode: 链码,参考fabric-samples/asset-transfer-private-data/chaincode-go
	- chaincode
		- smartcontract.go
	- go.mod
	- go.sum 
	- main.go
- data-sharing-webui: 前端/静态资源
- data-sharing-client: 客户端
- archive: 历史文件

- Fabric version: v2.5
- Golang version: v1.20

## Running

```bash
# create channel "mychannel" using ca
./network.sh up createChannel -c mychannel -ca

# deploy golang chaincode, setting endorsement policy and collection configuration
./network.sh deployCC -ccn [name] -ccp [path] -ccl go
```

1: `Asset` -> `Query`: `CreateAsset` -> `PutQuery`, `ReadAsset` -> `GetQuery`, `GetAllAssets` -> `GetAllQueries`
2: `TokenERC721Contract` -> `Card`

User ==> UserClient ==> ServiceClient ==> Chaincode & SQL

1. User select a service in webpage
2. UserClient starts a connection with ServiceClient
        1. UserClient request to start a connection
        2. ServiceClient send back a random message
        3. UserClient sign this message to verify the ownership of current account (i.e. peer node)
        4. ServiceClient verify the signed message with UserClient's public key and start listening
	- https://github.com/hyperledger/fabric-samples/blob/main/asset-transfer-events/application-gateway-go/app.go
3. ServiceClient resolve the request of user and invoke corredsponding chaincode

### Rest API

```bash
# start rest api
cd rest-api-go
go run main.go
```

POST:

```bash
curl --request POST \
  --url http://localhost:3000/invoke \
  --header 'content-type: application/x-www-form-urlencoded' \
  --data = \
  --data channelid=mychannel \
  --data chaincodeid=basic \
  --data function=createAsset \
  --data args=Asset123 \
  --data args=yellow \
  --data args=54 \
  --data args=Tom \
  --data args=13005
```

GET:

```bash
curl --request GET \
  --url 'http://localhost:3000/query?channelid=mychannel&chaincodeid=basic&function=ReadAsset&args=Asset123' 
```

## Basic Logics

- InitLedger
- !! CreateAsset
- !! ReadAsset
- ! UpdateAsset
- !! DeleteAsset
- ! AssetExists
- !! TransferAsset
- !! GetAllAssets

## Contract API

["github.com/hyperledger/fabric-contract-api-go"](https://pkg.go.dev/github.com/hyperledger/fabric-contract-api-go/contractapi)

["github.com/hyperledger/fabric-chaincode-go/shim"](https://pkg.go.dev/github.com/hyperledger/fabric-chaincode-go@v0.0.0-20240124143825-7dec3c7e7d45/shim) provides APIs for the chaincode to access its state variables, transaction context and call other chaincodes.

["github.com/hyperledger/fabric-chaincode-go/pkg/cid"](https://pkg.go.dev/github.com/hyperledger/fabric-chaincode-go/pkg/cid#section-readme) enables you to write chaincode which makes access control decisions based on the identity of the client (i.e. the invoker of the chaincode).

### Data


```go
// get public data
ctx.GetStub().GetState(id)

// get transient data
type assetTransientInput struct {
	...
}
var assetInput assetTransientInput
transientMap, err := ctx.GetStub().GetTransient()
if err != nil {
	return fmt.Errorf("error getting transient: %v", err)
}
transientAssetJSON, ok := transientMap["key"]
if !ok {
	return fmt.Errorf("asset not found in the transient map input")
}
err = json.Unmarshal(transientAssetJSON, &assetInput)
if err != nil {
	return fmt.Errorf("failed to unmarshal JSON: %v", err)
}

// get private data
ctx.GetStub().GetPrivateData("dataAssetCollection", id)
```

