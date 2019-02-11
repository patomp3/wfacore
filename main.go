package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

type appConfig struct {
	port     string
	queueurl string
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
		cfg.queueurl = viper.GetString(env + ".queueurl")

		log.Printf("## Loading Configuration")
		log.Printf("## Env\t= %s", env)
		log.Printf("## Port\t= %s", cfg.port)
	}

	router := mux.NewRouter()
	//router.HandleFunc("/test", test)
	router.HandleFunc("/submitorder", submitOrder).Methods("POST")
	router.HandleFunc("/getorder", getOrder)
	router.HandleFunc("/updateorder", updateOrder).Methods("POST")

	log.Printf("## Service Started....")

	log.Fatal(http.ListenAndServe(":"+cfg.port, router))
}

func getOrder(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("## Order Id = %s", orderID[0])
		// Get Order Id
		res = GetOrderService(orderID[0])
	}

	//log.Printf("%v", res)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func submitOrder(w http.ResponseWriter, r *http.Request) {

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

	log.Printf("## Request incoming...")
	log.Printf("## %v", req)

	//call recon api
	var res OrderResponse
	res = SubmitOrderService(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func updateOrder(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("## Request incoming...")
	log.Printf("## %v", req)

	//call update api
	var res UpdateResponse
	res = UpdateOrderService(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
