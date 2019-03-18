package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	smslog "github.com/patomp3/smslogs"
)

type appConfig struct {
	port     string
	queueurl string

	dbICC  string
	dbATB2 string
	dbPED  string

	env     string
	appName string
}

var cfg appConfig

func main() {

	log.Printf("##### Workflow Application - Core Service Started #####")

	// For no assign parameter env. using default to Test
	var env string
	if len(os.Args) > 1 {
		env = strings.ToLower(os.Args[1])
	} else {
		env = "development"
	}

	// Load configuration
	viper.SetConfigName("app")    // no need to include file extension
	viper.AddConfigPath("config") // set the path of your config file
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("## Config file not found. >> %s\n", err.Error())
	} else {
		// read config file
		cfg.port = viper.GetString(env + ".port")

		cfg.dbICC = viper.GetString(env + ".DBICC")
		cfg.dbATB2 = viper.GetString(env + ".DBATB2")
		cfg.dbPED = viper.GetString(env + ".DBPED")

		cfg.queueurl = viper.GetString(env + ".queueurl")
		cfg.env = viper.GetString(env + ".env")
		cfg.appName = viper.GetString("appName")

		log.Printf("## Loading Configuration")
		log.Printf("## System\t= %s", env)
		log.Printf("## Port\t= %s", cfg.port)
		log.Printf("## Env\t= %s", cfg.env)
	}

	router := mux.NewRouter()
	router.HandleFunc("/submitorder", submitOrder).Methods("POST")
	router.HandleFunc("/updateorder", updateOrder).Methods("POST")
	router.HandleFunc("/updatepayload", updatePayload).Methods("POST")
	router.HandleFunc("/getorder", getOrder)
	router.HandleFunc("/getpayload", getPayload)

	log.Printf("## Service Started....")

	log.Fatal(http.ListenAndServe(":"+cfg.port, router))

}

func getOrder(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n")
	//vars := mux.Vars(r)
	//orderID := vars["id"] // order id
	var res OrderResponse

	orderID, ok := r.URL.Query()["id"]
	if !ok {
		log.Println("## Invalid order id or missing value.")

		res.OrderTransID = ""
		res.ErrorCode = "1"
		res.ErrorDescription = "Invalid order id or missing value."

	} else {
		log.Printf("## Get Order Id = %s", orderID[0])
		// Get Order Id
		res = GetOrderService(orderID[0])
	}

	//log.Printf("%v", res)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func getPayload(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n")
	//vars := mux.Vars(r)
	//orderID := vars["id"] // order id
	var res OrderResponse

	orderID, ok := r.URL.Query()["id"]
	if !ok {
		log.Println("## Invalid order id or missing value.")

		res.OrderTransID = ""
		res.ErrorCode = "1"
		res.ErrorDescription = "Invalid order id or missing value."

	} else {
		log.Printf("## Get Payload for Order Id = %s", orderID[0])
		// Get Order Id
		res = GetPayloadService(orderID[0])
	}

	//submit log
	/*tvsNo := 0
	tvsRef := ""
	tvsOrder := orderID[0]
	jsonReq, _ := json.Marshal(orderID)
	jsonRes, _ := json.Marshal(res)
	submitLog("GetPayload", tvsNo, tvsRef, tvsOrder, string(jsonReq), string(jsonRes), res.ErrorCode, res.ErrorDescription)*/
	//End

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func updatePayload(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	//Read Json Request
	var req UpdatePayloadRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		panic(err)
	}

	log.Printf("## Request update payload incoming...")
	//log.Printf("## %v", req)

	//call update api
	var res UpdatePayloadResponse
	res = UpdatePayloadService(req)

	//submit log
	/*tvsNo := 0
	tvsRef := ""
	tvsOrder := req.OrderTransID
	jsonReq, _ := json.Marshal(req)
	jsonRes, _ := json.Marshal(res)
	submitLog("UpdatePayload", tvsNo, tvsRef, tvsOrder, string(jsonReq), string(jsonRes), res.ErrorCode, res.ErrorDescription)*/
	//End

	// Write log to stdoutput
	appFunc := "UpdatePayload"
	jsonReq, _ := json.Marshal(req)
	jsonRes, _ := json.Marshal(res)

	mLog := smslog.New(cfg.appName)
	mLog.OrderDate = ""
	mLog.OrderNo = ""
	mLog.OrderType = ""
	mLog.TVSNo = ""
	tag := []string{cfg.env, cfg.appName, appFunc, "INFO"}
	mLog.Tags = tag
	mLog.PrintLog(smslog.INFO, appFunc, req.OrderTransID, string(jsonReq), string(jsonRes))
	// End Write log

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func submitOrder(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	//Read Json Request
	var req OrderRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		panic(err)
	}

	log.Printf("## Request submit order incoming...")
	//log.Printf("## %v", req)

	//call recon api
	var res OrderResponse
	res = SubmitOrderService(req)

	//submit log to db
	/*tvsNo, _ := strconv.Atoi(req.TvsCustomerID)
	tvsRef := req.TvsReferenceID
	tvsOrder := ""
	jsonReq, _ := json.Marshal(req)
	jsonRes, _ := json.Marshal(res)
	submitLog("SubmitOrder", tvsNo, tvsRef, tvsOrder, string(jsonReq), string(jsonRes), res.ErrorCode, res.ErrorDescription)*/
	//End

	// Write log to stdoutput
	appFunc := "SubmitOrder"
	jsonReq, _ := json.Marshal(req)
	jsonRes, _ := json.Marshal(res)

	mLog := smslog.New(cfg.appName)
	mLog.OrderDate = req.RequestDate
	mLog.OrderNo = req.RequestTransID
	mLog.OrderType = req.OrderType
	mLog.TVSNo = req.TvsCustomerID
	tag := []string{cfg.env, cfg.appName, appFunc, "INFO"}
	mLog.Tags = tag
	mLog.PrintLog(smslog.INFO, appFunc, res.OrderTransID, string(jsonReq), string(jsonRes))
	// End Write log

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func updateOrder(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	//Read Json Request
	var req UpdateRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		panic(err)
	}

	log.Printf("## Request update order incoming...")
	//log.Printf("## %v", req)

	//call update api
	var res UpdateResponse
	res = UpdateOrderService(req)

	//submit log
	/*tvsNo := 0
	tvsRef := req.OrderID
	tvsOrder := req.OrderTransID
	jsonReq, _ := json.Marshal(req)
	jsonRes, _ := json.Marshal(res)
	submitLog("UpdateOrder", tvsNo, tvsRef, tvsOrder, string(jsonReq), string(jsonRes), res.ErrorCode, res.ErrorDescription)*/
	//End

	// Write log to stdoutput
	appFunc := "UpdateOrder"
	jsonReq, _ := json.Marshal(req)
	jsonRes, _ := json.Marshal(res)

	mLog := smslog.New(cfg.appName)
	mLog.OrderDate = ""
	mLog.OrderNo = ""
	mLog.OrderType = ""
	mLog.TVSNo = ""
	tag := []string{cfg.env, cfg.appName, appFunc, "INFO"}
	mLog.Tags = tag
	mLog.PrintLog(smslog.INFO, appFunc, req.OrderTransID, string(jsonReq), string(jsonRes))
	// End Write log

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

/*func submitLog(submitType string, tvsCustomerID int, tvsReferenceID string, orderID string,
	requestData string, responseData string, errCode string, errDesc string) {

	//#### Submit log
	dbPed := sms.New(cfg.dbPED)
	bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.InsertSubmitLog(:1,:2,:3,:4,:5,:6,:7,:8); end;",
		submitType, tvsCustomerID, tvsReferenceID, orderID, requestData, responseData, errCode, errDesc)
	if bResult {
		log.Printf("Submit log success.")
	} else {
		log.Printf("Submit log fail.")
	}
}*/
