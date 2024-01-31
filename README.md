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


## Private Data Logics

forked from asset-transfer-private-data

### Create Asset

1. Get new asset from transient map: `transientAssetJSON` -> Unmarshal -> `assetInput`
2. Check if asset already exists in private data "dataAssetCollection"
3. `verifyClientOrgMatchesPeerOrg(ctx)`
4. `asset` -> Marshal -> `assetJSONasBytes`
5. Put `assetJSONasBytes` into private data "dataAssetCollection"
6. `assetPrivateDetails` -> Marshal -> `assetPrivateDetailsAsBytes`
7. Put `assetPrivateDetailsAsBytes` into private data `getCollectionName(ctx)`

### Agree To Transfer

### Transfer Asset

### Delete Asset

### Purge Asset

PurgeAsset can be used by the owner of the asset to delete the asset. Trigger removal of the asset.

### Delete Tranfer Agreement

DeleteTranferAgreement can be used by the buyer to withdraw a proposal from the asset collection and from his own collection.

### Read Asset

### Read Asset Private Details

### Read Transfer Agreement

### Get Asset By Range

### Query AssetByOwner

### Query Assets

### Internal Functions

- verifyAgreement
- getCollectionName
- verifyClientOrgMatchesPeerOrg
- submittingClientIdentity
- getQueryResultForQueryString


## Research Example - `asset-transfer-basic`

- InitLedger
- !! CreateAsset
- !! ReadAsset
- ! UpdateAsset
- !! DeleteAsset
- AssetExists
- !! TransferAsset
- ! GetAllAssets


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


### 
