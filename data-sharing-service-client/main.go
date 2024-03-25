package main

import (
	"bytes"
	_ "context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	_ "errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/google/uuid"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	_ "github.com/hyperledger/fabric-gateway/pkg/identity"
	_ "github.com/hyperledger/fabric-protos-go-apiv2/gateway"
	_ "google.golang.org/grpc"
	_ "google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/status"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

const (
	channelName = "mychannel"
)

var (
	mspID        string
	cryptoPath   string
	certPath     string
	keyPath      string
	tlsCertPath  string
	peerEndpoint string
	gatewayPeer  string
)

type Service struct {
	ServiceName        string            `json:"ServiceName"`
	ServiceID          string            `json:"ServiceID"`
	PublisherURL       string            `json:"PublisherURL"`
	PublisherPublicKey string            `json:"PublisherPublicKey"`
	Comment            string            `json:"Comment"`
	ServiceHeaders     map[string]string `json:"ServiceHeaders"`
	TransactionHash    string            `json:"TransactionHash"`
}

type ServiceInfo struct {
	ServiceID     string `json:"ServiceID"`
	ServiceIP     string `json:"ServiceIP"`
	ServicePort   string `json:"ServicePort"`
	ServiceUser   string `json:"ServiceUser"`
	ServicePasswd string `json:"ServicePasswd"`
	ServiceDb     string `json:"ServiceDb"`
	ServiceTable  string `json:"ServiceTable"`
}

type Query struct {
	DataDigest     string `json:"DataDigest"`
	DataRows       int    `json:"DataRows"`
	InitiatorID    string `json:"InitiatorID"`
	InitiatorMSPID string `json:"InitiatorMSPID"`
	Legitimacy     bool   `json:"Legitimacy"`
	QueriedTable   string `json:"QueriedTable"`
	QueryDigest    string `json:"QueryDigest"`
	QueryID        string `json:"QueryID"`
	ServiceID      string `json:"ServiceID"`
	Timestamp      int    `json:"Timestamp"`
}

type ApplicationAnswer struct {
	InitiatorID  string `json:"InitiatorID"`
	InitiatorURL string `json:"InitiatorURL"`
	ServiceID    string `json:"ServiceID"`
	ServiceName  string `json:"ServiceName"`
	PublisherURL string `json:"PublisherURL"`
	// TODO
	ApplicationTime string `json:"ApplicationTime"`
	ProcessTime     string `json:"ProcessTime"`
	Status          int    `json:Status"` // 0-pending 1-approved 2-rejected
}

type Application struct {
	InitiatorURL       string           `json:"InitiatorURL"`
	InitiatorPublicKey *ecdsa.PublicKey `json:"InitiatorPublicKey"`
	InitiatorID        string           `json:"InitiatorID"`
	ServiceID          string           `json:"ServiceID"`
	ServiceName        string           `json:"ServiceName"`
	ApplicationTime    string           `json:"ApplicationTime"`
	Status             int              `json:"Status"`
}

// 定义 p256Curve 类型
type p256Curve struct {
	elliptic.CurveParams
}
type ResponseData struct {
	Data map[string]interface{} `json:"data"`
}

func init() {
	// 获取 P-256 曲线参数
	p256Params := elliptic.P256().Params()

	// 创建 p256Curve 实例
	p256Curve := p256Curve{*p256Params}

	// 注册 p256Curve 类型
	gob.Register(p256Curve)
}

var MyID string
var MyIP string
var MyURL string
var port string

func main() {
	// 从命令行读入端口
	if len(os.Args) < 2 {
		fmt.Println("Please provide a port number.")
		return
	}
	port = os.Args[1]
	fmt.Println("Port:", port)
	MyID = "User" + port
	MyIP, _ = getOuterIP()
	MyURL = "http://" + MyIP + ":" + port

	// 渲染页面
	app := gin.Default()
	app.LoadHTMLGlob("../data-sharing-webui/templates/*")

	type ViewData struct {
		Port string
	}

	app.GET("/login", func(c *gin.Context) {
		c.HTML(200, "login.html", nil)
	})

	app.GET("/", func(c *gin.Context) {
		data := ViewData{Port: port}
		c.HTML(200, "index.html", data)
	})

	fmt.Println("============= CONNECTING start =============")

	setConnParams(port)

	// connect to fabric
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()
	fmt.Println("======= Grpc Connection Established ========")

	id, MyPubKey := newIdentity() //得到一个新的id
	sign, signer := newSign()     //得到一个新的标记
	//	fmt.Println("cert: ", id)

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
	// chaincodeName := "basic" //初始化名字
	// if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
	// 	chaincodeName = ccname
	// } //如果ccname不等于“”，则将ccname赋值给chaincodename，

	//同上
	fmt.Println("============= Getting Contract =============")
	network := gw.GetNetwork("mychannel") //根据取得的channelName，调用GW的函数得到network,
	serviceChaincode := "ds_service"
	queryChaincode := "ds_query"
	serviceContract := network.GetContract(serviceChaincode) //得到contract
	queryContract := network.GetContract(queryChaincode)
	fmt.Println(serviceContract, queryContract)

	var applicationToMe []Application
	var myApplication []ApplicationAnswer
	var myService []ServiceInfo

	app.GET("/applicationToMe", func(c *gin.Context) {
		c.HTML(200, "applicationToMe.html", gin.H{
			"Port":         port,
			"applications": applicationToMe,
		})
	})

	app.GET("/myApplication", func(c *gin.Context) {
		// data := ViewData{Port: port}
		c.HTML(200, "myapplication.html", gin.H{
			"Port":         port,
			"applications": myApplication,
		})
	})

	// PUBLISHER
	// app.POST("/put_service", func(c *gin.Context) {
	// 	// 1. 接受前端传来的new_service
	// 	var new_service map[string]interface{}
	// 	if err := c.ShouldBindJSON(&new_service); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	// 2. 生成唯一的ServiceID
	// 	serviceID := uuid.New().String()
	// 	new_service["ServiceID"] = serviceID
	// 	new_service["PublisherURL"] = MyIP + ":" + string(port)
	// 	new_service["PublisherCertificate"] = id
	// 	fmt.Println("put_service generates new_service:",new_service)

	// 	// 3. 调用链码CreateAsset
	// 	response, err := serviceContract.SubmitTransaction("CreateService", new_service["ServiceName"], new_service["PublisherURL"], new_service["PublisherPublicKey"], new_service["Comment"])
	// 	if err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{
	// 			"transactionHash": nil,
	// 			"serviceID":       nil,
	// 			"error_msg":       fmt.Sprintf("Failed to submit transaction: %v", err),
	// 		})
	// 		return
	// 	}
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"transactionHash": response.TransactionID,
	// 		"serviceID":       serviceID,
	// 		"error_msg":       "None",
	// 	})

	// 	_, commit, err := serviceContract.SubmitAsync("CreateAsset", client.WithArguments()) // TODO
	// 	if err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{
	// 			"transactionHash": nil,
	// 			"serviceID":       nil,
	// 			"error_msg":       fmt.Sprintf("Failed to submit transaction: %v", err),
	// 		})
	// 		return
	// 	}
	// 	// 4. 根据是否调用成功，决定返回值
	// 	status, err := commit.Status()
	// 	if err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{
	// 			"transactionHash": nil,
	// 			"serviceID":       nil,
	// 			"error_msg":       fmt.Sprintf("Failed to get transaction commit status: %v", err),
	// 		})
	// 		return
	// 	}
	// 	if !status.Successful {
	// 		c.JSON(http.StatusOK, gin.H{
	// 			"transactionHash": nil,
	// 			"serviceID":       nil,
	// 			"error_msg":       fmt.Sprintf("Failed to commit transaction with status code %v", status.Code),
	// 		})
	// 		return
	// 	}
	// serviceInfo := ServiceInfo{
	// 	ServiceID:      serviceID,
	// 	ServiceIP:      new_service["IP"],
	// 	ServicePort:    new_service["Port"],
	// 	ServiceUser:    new_service["User"],
	// 	ServicePasswd:  new_service["Password"],
	// 	ServiceDb:      new_service["Database"],
	// 	ServiceTable:   new_service["Table"],
	// }
	// myService.append(serviceInfo)
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"transactionHash": commit.TransactionID(),
	// 		// "transactionHash": "testHash",
	// 		"serviceID": serviceID,
	// 		"error_msg": "None",
	// 	})
	// })

	app.GET("/get_services", func(c *gin.Context) {
		fmt.Println("===================== get_services start =====================")
		test_service := ServiceInfo{
			ServiceID:     "ID002",
			ServiceIP:     "localhost",
			ServicePort:   "3306",
			ServiceUser:   "root",
			ServicePasswd: "Rucsql_123",
			ServiceDb:     "test_db",
			ServiceTable:  "persons",
		}
		fmt.Println("In get_services, port is ", port)
		if port == "5001" {
			myService = append(myService, test_service)
			fmt.Println("get_services append", len(myService))
		}
		// result, err := serviceContract.EvaluateTransaction("GetAllServices")
		// if err != nil {
		// 	panic(fmt.Errorf("get_services failed to evoke chaincode: %w", err))
		// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		// 	return
		// }
		// var services []Service
		// err = json.Unmarshal(result, &services)
		// if err != nil {
		// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		// 	return
		// }

		//Test
		services := []Service{
			{
				ServiceName:        "Service1",
				ServiceID:          "ID001",
				PublisherURL:       "http://62.234.49.75:5000",
				PublisherPublicKey: "abc123PublicKey",
				Comment:            "Excellent service!",
				ServiceHeaders: map[string]string{
					"Authorization": "Bearer abc123",
					"Content-Type":  "application/json",
				},
				TransactionHash: "0x1234567892",
			},
			{
				ServiceName:        "Service2",
				ServiceID:          "ID002",
				PublisherURL:       "http://62.234.49.75:5001",
				PublisherPublicKey: "def456PublicKey",
				Comment:            "Reliable and efficient.",
				ServiceHeaders: map[string]string{
					"Authorization": "Bearer def456",
					"Content-Type":  "application/json",
				},
				TransactionHash: "0x1234567891",
			},
		}
		c.JSON(http.StatusOK, gin.H{"services": services})
	})

	app.GET("/get_toMe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"applications": applicationToMe})
	})

	app.GET("/get_sendOut", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"applications": myApplication})
	})

	app.Any("/send_application", func(c *gin.Context) {
		// 设置 CORS 头部
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		fmt.Println("send_application finish Header Set")

		if c.Request.Method == http.MethodOptions {
			// 处理预检请求的响应
			fmt.Println("send_application receive OPTIONS")
			c.Status(http.StatusOK)
			return
		}

		if c.Request.Method == http.MethodPost {
			fmt.Println("================= send_application start DEBUG ================")
			// fmt.Println("send_application receive referer: ", referer)
			body, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				fmt.Println("send_application read body err:", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			var application map[string]interface{}
			err = json.Unmarshal(body, &application)
			if err != nil {
				fmt.Println("send_application unmarshal response err: ", err)
				return
			}
			fmt.Println("send_application receive application:", application)
			InitiatorPublicKey := new(ecdsa.PublicKey)
			X := application["InitiatorPublicKeyX"].(string)
			Y := application["InitiatorPublicKeyY"].(string)
			InitiatorPublicKey.X, _ = new(big.Int).SetString(X, 16)
			InitiatorPublicKey.Y, _ = new(big.Int).SetString(Y, 16)
			InitiatorPublicKey.Curve = elliptic.P256()
			fmt.Println("initiatorpublickeyX: ", InitiatorPublicKey.X, "y:  ", InitiatorPublicKey.Y)
			// InitiatorPublicKey, err := decodePublicKey(encodedPubKey.([]byte))
			fmt.Println("send_application decode public Key", InitiatorPublicKey)
			// InitiatorPublicKey := application["InitiatorPublicKey"]
			referer := c.Request.Referer()
			newUrl := referer
			// 验签
			verified := execVerify(referer, InitiatorPublicKey)

			if verified {
				application["InitiatorURL"] = newUrl
				newApplication := Application{
					InitiatorPublicKey: InitiatorPublicKey,
					ApplicationTime:    time.Now().Format("2006-01-02 15:04:05"),
					InitiatorURL:       newUrl,
					InitiatorID:        application["InitiatorID"].(string),
					ServiceID:          application["ServiceID"].(string),
					ServiceName:        application["ServiceName"].(string),
				}
				fmt.Println("send_application successfully verify a signature.")
				applicationToMe = append(applicationToMe, newApplication)
				c.JSON(http.StatusOK, gin.H{"new_application": application})
			} else {
				fmt.Println("send_application failed in verifying a signature.")
				c.JSON(http.StatusBadRequest, nil)
			}
		}
	})

	// app.POST("/approve_application", func(c *gin.Context) {
	// 	var answer map[string]interface{}
	// 	if err := c.ShouldBindJSON(&answer); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	}
	// 	// 0. 获取Initiator的URL
	// 	InitiatorUrl := answer["InitiatorURL"].(string)
	// 	if answer["Status"] == 1 {
	// 		// 前端通过验证
	// 		// 1. 构造Query
	// 		NewQuery := Query{
	// 			DataDigest:     "ignored",
	// 			DataRows:       0,
	// 			InitiatorID:    answer["InitiatorID"].(string),
	// 			InitiatorMSPID: "ignored",
	// 			Legitimacy:     true,
	// 			QueriedTable:   "ignored",
	// 			QueryID:        "ignored",
	// 			QueryDigest:    "ignored",
	// 			ServiceID:      answer["ServiceID"].(string),
	// 			Timestamp:      int(time.Now().Unix())}
	// 		// 2. 调用链码，将Query上链
	// 		// TODO
	// 		_, commit, err := queryContract.SubmitAsync("PutQuery", client.WithArguments(NewQuery))
	// 		if err != nil {
	// 			panic(fmt.Errorf("failed to submit transaction: %w", err))
	// 		}
	// 		status, err := commit.Status()
	// 		if err != nil {
	// 			panic(fmt.Errorf("failed to get transaction commit status: %w", err))
	// 		}
	// 		if !status.Successful {
	// 			panic(fmt.Errorf("failed to commit transaction with status code %v", status.Code))
	// 		}
	// 	}
	// 	// 3. 向initiator的receive_answer路由发送审批结果
	// 	newUrl := InitiatorUrl + "/receive_answer"
	// 	var approved int
	// 	if answer["Status"] == 1 {
	// 		approved = 1
	// 	} else {
	// 		approved = 2
	// 	}
	// 	ret_answer := ApplicationAnswer{
	// 		InitiatorID:     answer["InitiatorID"].(string),
	// 		InitiatorURL:	 answer["InitiatorURL"].(string)
	// 		ServiceID:       answer["ServiceID"].(string),
	// 		ServiceName:	 answer["ServiceName"].(string),
	// 		PublisherURL:	 MyURL,
	// 		ProcessTime:     time.Now().Format("2006-01-02 15:04:05"),
	// 		Status:        approved,
	// 		ApplicationTime: answer["ApplicationTime"],
	// 	}
	// 	payload := map[string]interface{}{
	// 		"ApplicationAnswer": ret_answer,
	// 	}
	// 	body, err := json.Marshal(payload)
	// 	if err != nil {
	// 		fmt.Println("approve_application json.Marshal err: ", err)
	// 		return
	// 	}
	// 	req, err := http.NewRequest(http.MethodPost, newUrl, bytes.NewReader(body))
	// 	if err != nil {
	// 		fmt.Println("approve_application http.NewRequest err: ", err)
	// 		return
	// 	}
	// 	req.Header.Set("Content-Type", "application/json")
	// 	httpClient := &http.Client{Timeout: 5 * time.Second}
	// 	resp, err := httpClient.Do(req)
	// 	defer resp.Body.Close()
	// 	if err != nil {
	// 		fmt.Println("approve_application http.DefaultClient.Do() err:", err)
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 		return
	// 	}

	// 	// 4. 删除applicationToMe里的对应项
	// 	for i := 0; i < len(applicationToMe); i++ {
	// 		if applicationToMe[i].InitiatorID == answer["InitiatorID"].(string) && applicationToMe[i].ServiceID == answer["ServiceID"].(string) {
	// 			applicationToMe = append(applicationToMe[:i], applicationToMe[i+1:]...)
	// 			break
	// 		}
	// 	}
	// })

	app.POST("/request_data", func(c *gin.Context) {
		fmt.Println("=============================== request_data start ===============================")
		// 接收从Initiator发来的数据申请，验证其权限后返回
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		fmt.Println("request_data finish Header Set")

		// 1. 解析申请数据包
		body, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			fmt.Println("send_application read body err:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var application map[string]interface{}
		err = json.Unmarshal(body, &application)
		if err != nil {
			fmt.Println("request_data json.Unmarshal err:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 1.1 验签
		InitiatorPublicKey := new(ecdsa.PublicKey)
		X := application["InitiatorPublicKeyX"].(string)
		Y := application["InitiatorPublicKeyY"].(string)
		InitiatorPublicKey.X, _ = new(big.Int).SetString(X, 16)
		InitiatorPublicKey.Y, _ = new(big.Int).SetString(Y, 16)
		InitiatorPublicKey.Curve = elliptic.P256()

		referer := c.Request.Referer()
		verified := execVerify(referer, InitiatorPublicKey)

		if !verified {
			fmt.Println("request_data failed in verifying a signature.")
			c.JSON(http.StatusBadRequest, nil)
			return
		}
		// 2. 调用链码
		// TODO!!!!!
		// 3. 根据链码结果决定向Initiator的fetch_data发什么内容
		// _, commit, err := queryContract.SubmitAsync("PutQuery", client.WithArguments("certificate", "id", "dataDigest", -1, "initiatorID", "initiatorMSPID", false, "queriedTable", "queryDigest", "serviceID", 0))
		// if err != nil {
		// 	panic(fmt.Errorf("failed to submit transaction: %w", err))
		// }
		// status, err := commit.Status()
		// if err != nil {
		// 	panic(fmt.Errorf("failed to get transaction commit status: %w", err))
		// }
		// if !status.Successful {
		// 	panic(fmt.Errorf("failed to commit transaction with status code %v", status.Code))
		// }
		// 3 从数据库获取data
		// 3.1 找到对应ServiceInfo
		var info ServiceInfo
		serviceID := application["ServiceID"]
		found := false
		fmt.Println("myService length:", len(myService))
		for i := 0; i < len(myService); i++ {
			fmt.Println("serviceInfo search:", myService[i])
			if myService[i].ServiceID == serviceID {
				info = myService[i]
				found = true
				break
			}
		}
		if !found {
			fmt.Println("request_data failed in finding a service.")
			c.JSON(http.StatusBadRequest, nil)
			return
		}
		// 3.2 用info里的信息调用select
		data := dataBase(info.ServiceUser, info.ServicePasswd, info.ServiceIP, info.ServicePort, info.ServiceDb, info.ServiceTable)
		fmt.Println("request_data get data:", data)
		c.JSON(http.StatusOK, gin.H{"data": data})
	})

	// INITIATOR
	app.POST("/forward_application", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		newApplication := ApplicationAnswer{
			InitiatorID:     MyID,
			InitiatorURL:    MyURL,
			ServiceID:       data["ServiceID"].(string),
			ServiceName:     data["ServiceName"].(string),
			PublisherURL:    data["PublisherURL"].(string),
			Status:          0,
			ApplicationTime: time.Now().Format("2006-01-02 15:04:05")}
		myApplication = append(myApplication, newApplication)
		PublisherURL := data["PublisherURL"].(string)
		data["InitiatorID"] = MyID
		data["InitiatorURL"] = MyURL
		// encode pubKey
		// encodedPubKey, err := encodePublicKey(MyPubKey)
		// if err != nil {
		// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		// 	return
		// }
		//data["InitiatorPublicKeyCurve"] = MyPubKey.Curve.Params()
		data["InitiatorPublicKeyX"] = MyPubKey.X.Text(16)
		data["InitiatorPublicKeyY"] = MyPubKey.Y.Text(16)

		sendData, err := json.Marshal(data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		fmt.Println("======================== forward_application sending data ========================")
		fmt.Println("data: ", data)
		fmt.Println("objURL:", PublisherURL+"/send_application")
		fmt.Println("MyIP:", MyIP)

		req, err := http.NewRequest("POST", PublisherURL+"/send_application", bytes.NewBuffer(sendData))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		refererURL := MyURL
		// refererURL, err := url.Parse((MyIP + ":" + port))
		// if err != nil {
		// 	fmt.Println("forward_application parsing url error: ", err)
		// 	return
		// }
		req.Header.Set("Referer", refererURL)
		fmt.Println("forward_application set header:", refererURL)

		httpClient := &http.Client{}
		res, err := httpClient.Do(req)
		if err != nil {
			fmt.Println("forward_application err on httpClient.Do()", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fmt.Println("======================== forward_application receive response ========================")
		fmt.Println("res content: ", res)
		c.JSON(200, gin.H{"success": "success"})
		defer res.Body.Close()

		// services, err := ioutil.ReadAll(res.Body)
		// if err != nil {
		// 	fmt.Println("forward_application can't receive services: ", err.Error())
		// c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		// return
		// }

		// var service Service
		// for i range in services {
		// 	if i["ServiceID"]==serviceID{
		// 		service = i
		// 		break
		// 	}
		// }
		// content := map[string]interface{}{
		// 	"buyerId" : service["buyerID"],
		// 	"buyerPublicKey" : service["buyerPublicKey"],
		// 	"applicationTime" : service["applicationTime"],
		// 	"nounce" : service["nounce"],
		// }
		// sign := ourSign(getPrivateKey(),[]byte(content))
		// content["sign"]=string(sign)
		// sendData,_ := json.Marshal(content)
		// res, err := http.Post(service["sellerurl"]+"/send_application",
		// "application/json", bytes.NewBuffer([]byte(sendData)))

		// if err != nil {
		// 	fmt.Println("Fatal error ", err.Error())
		// }else{
		// 	c.JSON(200,gin.H{
		// 		"applicationID",ioutil.ReadAll(res.Body),
		// 	})
		// }
	})

	app.POST("/fetch_data", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		data["InitiatorID"] = MyID
		data["InitiatorURL"] = MyURL
		data["InitiatorPublicKeyX"] = MyPubKey.X.Text(16)
		data["InitiatorPublicKeyY"] = MyPubKey.Y.Text(16)
		// serviceID := data["ServiceID"]
		PublisherURL := data["PublisherURL"].(string)

		sendData, err := json.Marshal(data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		req, err := http.NewRequest("POST", PublisherURL+"/request_data", bytes.NewBuffer(sendData))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		refererURL := MyURL
		req.Header.Set("Referer", refererURL)

		httpClient := &http.Client{}
		res, err := httpClient.Do(req)
		if err != nil {
			fmt.Println("forward_application err on httpClient.Do()", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer res.Body.Close()

		fmt.Println("======================== fetch_data receive response ========================")

		// TODO: decode message
		respBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("fetch_data err on ioutil.ReadAll()", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var respData map[string]interface{}
		err = json.Unmarshal(respBody, &respData)
		if err != nil {
			fmt.Println("fetch_data err on json.Unmarshal()", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		retdata := respData["data"]
		fmt.Println("fetch_data get data: ", retdata)
		c.JSON(200, gin.H{"data": retdata})

	})

	app.POST("/receive_message", func(c *gin.Context) {
		// 1. receive raw message
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		fmt.Println("============= receive_message start debug =============")
		fmt.Println("receive_message receive message", data)
		// 2. generate the signature
		message := data["message"]
		signature, err := SignMessage(message.(string), signer)
		if err != nil {
			fmt.Println("receive_message SignMessage() err", err)
		}
		// content := map[string]interface{}{
		// 	"message": message,
		// 	"sign":    signature,
		// }
		// 3. send in response
		// sendData, _ := json.Marshal(content)
		// res, err := http.Post(service["sellerurl"]+"/receive_sign",
		// "application/json", bytes.NewBuffer([]byte(sendData)))
		//返回值
		c.JSON(http.StatusOK, gin.H{"message": message, "signature": signature})
	})

	app.POST("/receive_answer", func(c *gin.Context) {
		// 修改myApplication里对应列表的Status
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for i := 0; i < len(myApplication); i++ {
			if myApplication[i].ServiceID == data["ServiceID"].(string) && myApplication[i].InitiatorID == data["InitiatorID"].(string) {
				myApplication[i].Status = data["Status"].(int)
				// update applicationtime
				myApplication[i].ApplicationTime = data["ApplicationTime"].(string)
			}
			break
		}
	})

	app.GET("/app.js", func(c *gin.Context) {
		c.File("../data-sharing-webui/app.js")
	})
	app.GET("/toMe.js", func(c *gin.Context) {
		c.File("../data-sharing-webui/toMe.js")
	})
	app.GET("/sendOut.js", func(c *gin.Context) {
		c.File("../data-sharing-webui/sendOut.js")
	})

	app.GET("/styles.css", func(c *gin.Context) {
		c.File("../data-sharing-webui/styles.css")
	})

	// 将 "../data-sharing-webui" 目录下的静态文件映射到 "/static" 路由路径
	// app.Use(static.Serve("/static", static.LocalFile("../data-sharing-webui", true)))
	app.Static("/static", "../data-sharing-webui")

	// 监听并启动 5000 端口
	portStr := string(port)
	app.Run(":" + portStr)
}

func getOuterIP() (ipv4 string, err error) {
	resp, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(ip)), nil
}

func GetLocalPort() (string, error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer ln.Close()

	port := fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
	return port, nil
}

func setConnParams(port string) {
	if port == "5000" {
		mspID = "Org1MSP"
		cryptoPath = "/home/ubuntu/hyperledger/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com"
		certPath = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
		keyPath = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
		tlsCertPath = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
		peerEndpoint = "localhost:7051"
		gatewayPeer = "peer0.org1.example.com"
	} else if port == "5001" {
		mspID = "Org2MSP"
		cryptoPath = "/home/ubuntu/hyperledger/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com"
		certPath = cryptoPath + "/users/User1@org2.example.com/msp/signcerts/cert.pem"
		keyPath = cryptoPath + "/users/User1@org2.example.com/msp/keystore/"
		tlsCertPath = cryptoPath + "/peers/peer0.org2.example.com/tls/ca.crt"
		peerEndpoint = "localhost:9051"
		gatewayPeer = "peer0.org2.example.com"
	} else {
		mspID = "Org1MSP"
		cryptoPath = "/home/ubuntu/hyperledger/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com"
		certPath = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
		keyPath = cryptoPath + "/users/User1@org1.example.com/msp/keystore/"
		tlsCertPath = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
		peerEndpoint = "localhost:7051"
		gatewayPeer = "peer0.org1.example.com"
	}
}

func generateRandomMessage() string {
	b := make([]byte, 10)
	rand.Read(b)
	randomMessage := fmt.Sprintf("%x", b)
	return randomMessage
}

func SignMessage(message string, signer crypto.Signer) (string, error) {
	hash := sha256.Sum256([]byte(message))

	signature, err := signer.Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	signatureStr := base64.StdEncoding.EncodeToString(signature)
	return signatureStr, nil
}

func VerifySignature(message, signature string, publicKey *ecdsa.PublicKey) error {
	decodedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	hash := sha256.Sum256([]byte(message))

	// 将 *ecdsa.PublicKey 转换为 crypto.PublicKey
	if !ecdsa.VerifyASN1(publicKey, hash[:], decodedSignature) {
		return fmt.Errorf("invalid signature")
	}
	fmt.Println("VerifySignature successfully verify a signature")
	return nil
}

func dataBase(usrname string, passwd string, ip string, port string, databaseName string, tableName string) []map[string]interface{} {
	dsn := usrname + ":" + passwd + "@tcp(" + ip + ":" + port + ")/" + databaseName
	db, err := sql.Open("mysql", dsn)
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
	sqlStr := "select * from " + tableName + ";"
	rows, err := db.Query(sqlStr)
	if err != nil {
		fmt.Println(err)
	}
	// defer close result set
	defer rows.Close()

	cols, _ := rows.Columns()
	var ret []map[string]interface{}
	if len(cols) > 0 {
		for rows.Next() {
			buff := make([]interface{}, len(cols))
			data := make([][]byte, len(cols)) //数据库中的NULL值可以扫描到字节中
			for i := range buff {
				buff[i] = &data[i]
			}
			rows.Scan(buff...) //扫描到buff接口中，实际是字符串类型data中
			//将每一行数据存放到数组中
			dataKv := make(map[string]interface{}, len(cols))
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

func execVerify(referer string, InitiatorPublicKey *ecdsa.PublicKey) bool {
	// 1. 生成随机message
	randomMessage := generateRandomMessage()
	fmt.Println("execVerify generate random message: ", randomMessage)

	// 2. 将随机message发送给申请方的receive_message接口
	// 2.1 获取对方的url
	newUrl := referer + "/receive_message"
	// 2.2 构造payload
	payload := map[string]interface{}{
		"message":      randomMessage,
		"PublisherUrl": MyURL,
	}
	fmt.Println("execVerify generate payload: ", payload)
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("execVerify json.Marshal err: ", err)
		return false
	}
	// 2.3 send message
	req, err := http.NewRequest(http.MethodPost, newUrl, bytes.NewReader(body))
	if err != nil {
		fmt.Println("execVerify http.NewRequest err: ", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println("execVerify http.DefaultClient.Do() err:", err)
		// c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return false
	}
	defer resp.Body.Close()

	// 4.等待签名后的结果
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("execVerify ioutil.ReadAll() err: ", err)
		return false
	}

	var respData map[string]interface{}
	err = json.Unmarshal(respBody, &respData)
	fmt.Println("respData:", respData)
	if err != nil {
		fmt.Println("execVerify unmarshal response err: ", err)
		// c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return false
	}

	signedMessage, ok := respData["signature"].(string)
	if !ok {
		// c.JSON(http.StatusBadRequest, gin.H{"error": "invalid response format"})
		return false
	}

	// 5. 验证签名
	fmt.Println("execVerify initiator pub key:", InitiatorPublicKey)
	// InitPubKey, err := parseECDSAPublicKey(InitiatorPublicKey)
	if err != nil {
		fmt.Println("send_application parseECDSAPublicKey err: ", err)
		// c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}
	verifiedErr := VerifySignature(randomMessage, signedMessage, InitiatorPublicKey)
	if verifiedErr == nil {
		return true
	} else {
		return false
	}
}

// // parseECDSAPublicKey 解析 ECDSA 公钥
// func parseECDSAPublicKey(pubKeyMap map[string]interface{}) (*ecdsa.PublicKey, error) {
// 	// 从 map 中提取曲线参数
// 	curveName, ok := pubKeyMap["Curve"].(string)
// 	if !ok {
// 		return nil, fmt.Errorf("invalid curve type")
// 	}
// 	curve := elliptic.GetCurveByName(curveName)
// 	if curve == nil {
// 		return nil, fmt.Errorf("unsupported curve: %s", curveName)
// 	}

// 	// 从 map 中提取 X 和 Y 坐标值
// 	xValue, ok := pubKeyMap["X"].(float64)
// 	if !ok {
// 		return nil, fmt.Errorf("invalid X value type")
// 	}
// 	yValue, ok := pubKeyMap["Y"].(float64)
// 	if !ok {
// 		return nil, fmt.Errorf("invalid Y value type")
// 	}

// 	// 构造 ecdsa.PublicKey 对象
// 	x := big.NewFloat(xValue)
// 	y := big.NewFloat(yValue)
// 	xInt, _ := x.Int(nil)
// 	yInt, _ := y.Int(nil)
// 	pubKey := ecdsa.PublicKey{
// 		Curve: curve,
// 		X:     xInt,
// 		Y:     yInt,
// 	}

// 	return &pubKey, nil
// }

/*
func encodePublicKey(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.MarshalCompressed(elliptic.P256(), pub.X, pub.Y)
}

func decodePublicKey(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), pub)
	if x == nil {
		error := fmt.Errorf("invalid public key")
		return nil, error
	}
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, nil
}*/

/*func encodePublicKey(pubKey *ecdsa.PublicKey) ([]byte, error) {
	var buf bytes.Buffer
	gob.Register(elliptic.P256())
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(pubKey)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// 从字节切片解码公钥
func decodePublicKey(data []byte) (*ecdsa.PublicKey, error) {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var pubKey ecdsa.PublicKey
	err := dec.Decode(&pubKey)
	if err != nil {
		return nil, err
	}
	return &pubKey, nil
}
*/
