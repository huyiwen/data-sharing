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

	"database/sql"
	 _ "github.com/go-sql-driver/mysql" //导入包但不使用，init()
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

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
	//接收的数据：
	//headers 
	app.POST("/fetch_data", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		headers := data["headers"]
		serviceID := data["serviceID"]
		res,err := http.POST(service["sellerurl"]+"/get_services","application/json",nil)//待完善
		if err != nil {
			panic(err)
		}
		services, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Fatal error ", err.Error())
			panic(err)
		}

		var service Service
		for i range in services {
			if i["serviceid"]==serviceID{
				service = i
				break
			}
		}
		currenttime := time.Now()
		payload := make(map[string]interface{}){
			"headers" : headers,
			"currentTime" : currenttime,
		}
		sendData,_ := json.Marshal(content)
		res, err := http.Post(service["sellerurl"]+"/request_data",
		"application/json", bytes.NewBuffer([]byte(sendData)))

		c.JSON(http.StatusOK, gin.H{
			//"data":         ,
			"column_names": headers,
		})
	})

	app.POST("/forward_application",func(c *gin.Context){
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		serviceID := data["serviceID"]
		res,err := http.POST(service["sellerurl"]+"/get_services","application/json",nil)//待完善
		if err != nil {
			panic(err)
		}
		services, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Fatal error ", err.Error())
			panic(err)
		}

		var service Service
		for i range in services {
			if i["serviceid"]==serviceID{
				service = i
				break
			}
		}
		content := make(map[string]interface{}){
			"buyerId" : service["buyerID"],
			"buyerPublicKey" : service["buyerPublicKey"],
			"applicationTime" : service["applicationTime"],
			"nounce" : service["nounce"],
		}
		sign := ourSign(getPrivateKey(),[]byte(content))
		content["sign"]=string(sign)
		sendData,_ := json.Marshal(content)
		res, err := http.Post(service["sellerurl"]+"/send_application",
		"application/json", bytes.NewBuffer([]byte(sendData)))

		if err != nil {
			fmt.Println("Fatal error ", err.Error())
		}else{
			c.JSON(200,gin.H{
				"applicationID",ioutil.ReadAll(res.Body),
			})
		}
	})
	
	app.POST("/receive_message",func(c *gin.Context){
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sellerurl:=data["sellerurl"]
		message:=data["message"]
		sign:=ourSign(getPrivateKey(),message)
		content:=make(map[string]interface){
			"message":message,
			"sign":sign,
		}
		sendData,_:=json.Marshal(content)
		res, err := http.Post(service["sellerurl"]+"/receive_sign",
		"application/json", bytes.NewBuffer([]byte(sendData)))
		//返回值
		c.JSON(200,gin.H{"sign": sign})
	})

	app.POST("/receive_applicationAnswer",func(c *gin.Context){
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK,gin.H{
			"serviceID" : data["ServiceID"],
			"initiatorID" : data["InitiatorID"],
			"legitimacy" : data[legitimacy],
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

//获取私钥
func getPrivateKey() crypto.privateKey{
	files, err := os.ReadDir(keyPath) //读取目录
	if err != nil {
		panic(fmt.Errorf("failed to read private key directory: %w", err))
		return nil
	}
	privateKeyPEM, err := os.ReadFile(path.Join(keyPath, files[0].Name())) //得到私钥pem格式

	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
		return nil
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM) //根据pem得到私钥
	if err != nil {
		panic(err)
		return nil
	}
	return privateKey
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
// newSign 创建一个函数，使用私钥从信息摘要生成数字签名.
func newSign() identity.Sign {
	privatekey := getPrivateKey()
	if privatekey == nil(
		return nil
	)

	sign, err := identity.NewPrivateKeySign(privateKey) //根据私钥得到数字签名
	if err != nil {
		panic(err)
	}

	return sign
}
func ourSign(privateKey ed25519.PrivateKey, message []byte) Sign {
		signature := ed25519.Sign(privateKey, message)
		return signature, nil
}
func dataBase(usrname string, passwd string, id string, databaseName string, tableName string) []map[interface]string{
	dsn := usrname+passwd+id+databaseName
	db,err := sql.Open("mysql",dsn)
	if err != nil {
		fmt.Printf("dsn:%s invalid,err:%v\n", dsn, err)
		return nil
	}
	defer db.Close()
	err = db.Ping() //尝试连接数据库
	if err != nil {
		fmt.Printf("open %s faild,err:%v\n", dsn, err)
		return nil
	}
	sqlStr := "select * from ? ;"
	rows,err := db.Query(sqlStr,tableName);
	if err != nil {
        fmt.Println(err)
    }
    // defer close result set
    defer rows.Close()

   if len(cols) > 0 {
      var ret []map[string]string
      for rows.Next() {
         buff := make([]interface{}, len(cols))
         data := make([][]byte, len(cols)) //数据库中的NULL值可以扫描到字节中
         for i, _ := range buff {
            buff[i] = &data[i]
         }
         rows.Scan(buff...) //扫描到buff接口中，实际是字符串类型data中
         //将每一行数据存放到数组中
         dataKv := make(map[string]string, len(cols))
         for k, col := range data { //k是index，col是对应的值
            //fmt.Printf("%30s:\t%s\n", cols[k], col)
            dataKv[cols[k]] = string(col)
         }
         ret = append(ret, dataKv)
      }
      return ret
   } else {
      return nil
   }
}