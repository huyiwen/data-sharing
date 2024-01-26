package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	mspID        = "Org1MSP"
	cryptoPath   = "../../test-network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
)

type Service struct {
	ServiceName      string            `json:"serviceName"`
	ServiceID        string            `json:"serviceID"`
	SellerURL        string            `json:"sellerURL"`
	SellerPublicKey  string            `json:"sellerPublicKey"`
	Comment          string            `json:"comment"`
	SellerHeaders    map[string]string `json:"sellerHeaders"`
	TransactionHash  string            `json:"transactionHash"`
}

var now = time.Now() //得到当前时间
var assetId = fmt.Sprintf("asset%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)

func main() {
	clientConnection := newGrpcConnection() //建立一个新的grpc连接
	defer clientConnection.Close()
	id := newIdentity() //得到一个新的id
	sign := newSign()   //得到一个新的标记

	gw, err := client.Connect( //新建一个客户端连接并且进行初始化
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close() //关闭gateway

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "basic" //初始化名字
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	} //如果ccname不等于“”，则将ccname赋值给chaincodename，

	//同上
	network := gw.GetNetwork(channelName)          //根据取得的channelName，调用GW的函数得到network,
	contract := network.GetContract(chaincodeName) //得到contract

	app := gin.Default()

	app.LoadHTMLGlob("templates/*.html")

	app.GET("/", func(c *gin.Context) {
		c.HTML(200,"index.html",nil)
	})

	app.POST("/get_services", func(c *gin.Context) {
		services := []Service{
			{
				ServiceName:     "Service1",
				ServiceID:       "ID001",
				SellerURL:       "http://seller1.example.com",
				SellerPublicKey: "abc123PublicKey",
				Comment:         "Excellent service!",
				SellerHeaders: map[string]string{
					"Authorization": "Bearer abc123",
					"Content-Type":  "application/json",
				},
				TransactionHash: "0x1234567892",
			},
			{
				ServiceName:     "Service2",
				ServiceID:       "ID002",
				SellerURL:       "http://seller2.example.com",
				SellerPublicKey: "def456PublicKey",
				Comment:         "Reliable and efficient.",
				SellerHeaders: map[string]string{
					"Authorization": "Bearer def456",
					"Content-Type":  "application/json",
				},
				TransactionHash: "0x1234567891",
			},
			{
				ServiceName:     "Service3",
				ServiceID:       "ID003",
				SellerURL:       "http://seller3.example.com",
				SellerPublicKey: "ghi789PublicKey",
				Comment:         "Good, but room for improvement.",
				SellerHeaders: map[string]string{
					"Authorization": "Bearer ghi789",
					"Content-Type":  "application/json",
				},
				TransactionHash: "0x1234567890",
			},
		}

		c.JSON(http.StatusOK, gin.H{"services": services})
	})

	app.POST("/send_application",func(c *gin.Context){

	})

	app.POST("/approve_application", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Process data...

		c.JSON(http.StatusOK, gin.H{"serviceID": "ID005"})
	})

	app.POST("/put_service", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Process data...

		c.JSON(http.StatusOK, gin.H{"serviceID": "ID005", "transactionHash": "0x123468q235"})
	})

	app.POST("/fetch_data", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Process data...

		c.JSON(http.StatusOK, gin.H{
			"data":         [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
			"column_names": []string{"column1", "column2", "column3"},
		})
	})

	app.GET("/app.js", func(c *gin.Context) {
		c.File("../app.js")
	})

	app.GET("/styles.css", func(c *gin.Context) {
		c.File("../styles.css")
	})

	app.Static("/static","./static")

	// 监听并启动 5000 端口
	app.Run(":5000")
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
// 建立一个新的grpc连接到gateway服务端
func newGrpcConnection() *grpc.ClientConn {
	certificate, err := loadCertificate(tlsCertPath) //根据常量来下载一个证书
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()                                                  //证书池？
	certPool.AddCert(certificate)                                                   //添加证书
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer) //运输证书，调用golang的包，(感觉弄一个证书池有些不明白)

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials)) //开启连接
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
// newIdentity 使用 X.509 证书为网关连接创建客户端身份。
func newIdentity() *identity.X509Identity { //得到新的id
	certificate, err := loadCertificate(certPath) //定义一个证书
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate) //根据刚刚新创立的证书调用函数得到id
	if err != nil {
		panic(err)
	}

	return id
}

// 下载证书，以x509.certificate的格式存储
func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := os.ReadFile(filename) //读取文件里面的内容
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}
	return identity.CertificateFromPEM(certificatePEM) //应该是一一个解析函数，将证书里面的内容存储到x509.certificate里面
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
// newSign 创建一个函数，使用私钥从信息摘要生成数字签名。
func newSign() identity.Sign {
	files, err := os.ReadDir(keyPath) //读取目录
	if err != nil {
		panic(fmt.Errorf("failed to read private key directory: %w", err))
	}
	privateKeyPEM, err := os.ReadFile(path.Join(keyPath, files[0].Name())) //得到私钥pem格式

	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM) //根据pem得到私钥
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey) //根据私钥得到数字签名
	if err != nil {
		panic(err)
	}

	return sign
}

// This type of transaction would typically only be run once by an application the first time it was started after its
// initial deployment. A new version of the chaincode deployed later would likely not need to run an "init" function.
// 这种类型的事务通常只在应用程序首次启动后运行一次。运行一次。之后部署的新版本 chaincode 可能不需要运行 "init "函数。
// 初始化账册
func initLedger(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: InitLedger, function creates the initial set of assets on the ledger \n")

	_, err := contract.SubmitTransaction("InitLedger") //提交交易
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Evaluate a transaction to query ledger state.
// 评估一个交易来查询分类帐状态
func getAllAssets(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: GetAllAssets, function returns all the current assets on the ledger")

	evaluateResult, err := contract.EvaluateTransaction("GetAllAssets") //调用contract的函数，得到评估结果，
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult) //把结果改成字符串类型

	fmt.Printf("*** Result:%s\n", result)
}

// Submit a transaction synchronously, blocking until it has been committed to the ledger.
// 同步提交事务，在事务提交到分类账之前一直处于阻塞状态。
func createAsset(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: CreateAsset, creates new asset with ID, Color, Size, Owner and AppraisedValue arguments \n")

	_, err := contract.SubmitTransaction("CreateAsset", assetId, "yellow", "5", "Tom", "1300") //合约里面上传一个新的合约，里面是初始化参数
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Evaluate a transaction by assetID to query ledger state.
// 按资产 ID 评估交易，以查询分类账状态。
func readAssetByID(contract *client.Contract) {
	fmt.Printf("\n--> Evaluate Transaction: ReadAsset, function returns asset attributes\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadAsset", assetId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Submit transaction asynchronously, blocking until the transaction has been sent to the orderer, and allowing
// this thread to process the chaincode response (e.g. update a UI) without waiting for the commit notification
// 异步提交事务，阻塞直到事务被发送给提交者，并允许该线程处理链码响应（例如更新用户界面），而无需等待提交通知
func transferAssetAsync(contract *client.Contract) {
	fmt.Printf("\n--> Async Submit Transaction: TransferAsset, updates existing asset owner")

	submitResult, commit, err := contract.SubmitAsync("TransferAsset", client.WithArguments(assetId, "Mark")) //异步提交
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction asynchronously: %w", err))
	}

	fmt.Printf("\n*** Successfully submitted transaction to transfer ownership from %s to Mark. \n", string(submitResult))
	fmt.Println("*** Waiting for transaction commit.")

	if commitStatus, err := commit.Status(); err != nil {
		panic(fmt.Errorf("failed to get commit status: %w", err))
	} else if !commitStatus.Successful {
		panic(fmt.Errorf("transaction %s failed to commit with status: %d", commitStatus.TransactionID, int32(commitStatus.Code)))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Submit transaction, passing in the wrong number of arguments ,expected to throw an error containing details of any error responses from the smart contract.
func exampleErrorHandling(contract *client.Contract) { //错误处理
	fmt.Println("\n--> Submit Transaction: UpdateAsset asset70, asset70 does not exist and should return an error")

	_, err := contract.SubmitTransaction("UpdateAsset", "asset70", "blue", "5", "Tomoko", "300")
	if err == nil {
		panic("******** FAILED to return an error")
	}

	fmt.Println("*** Successfully caught the error:")

	switch err := err.(type) {
	case *client.EndorseError:
		fmt.Printf("Endorse error for transaction %s with gRPC status %v: %s\n", err.TransactionID, status.Code(err), err)
	case *client.SubmitError:
		fmt.Printf("Submit error for transaction %s with gRPC status %v: %s\n", err.TransactionID, status.Code(err), err)
	case *client.CommitStatusError:
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("Timeout waiting for transaction %s commit status: %s", err.TransactionID, err)
		} else {
			fmt.Printf("Error obtaining commit status for transaction %s with gRPC status %v: %s\n", err.TransactionID, status.Code(err), err)
		}
	case *client.CommitError:
		fmt.Printf("Transaction %s failed to commit with status %d: %s\n", err.TransactionID, int32(err.Code), err)
	default:
		panic(fmt.Errorf("unexpected error type %T: %w", err, err))
	}

	// Any error that originates from a peer or orderer node external to the gateway will have its details
	// embedded within the gRPC status error. The following code shows how to extract that.
	statusErr := status.Convert(err)

	details := statusErr.Details()
	if len(details) > 0 {
		fmt.Println("Error Details:")

		for _, detail := range details {
			switch detail := detail.(type) {
			case *gateway.ErrorDetail:
				fmt.Printf("- address: %s, mspId: %s, message: %s\n", detail.Address, detail.MspId, detail.Message)
			}
		}
	}
}

// Format JSON data
// 格式化json数据，应该是转为string
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}