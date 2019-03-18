package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"log"
	"strconv"

	"github.com/spf13/viper"

	sms "github.com/patomp3/smsservices"
	pubModule "github.com/patomp3/wfacore/module"
)

// UpdateRequest for ...
type UpdateRequest struct {
	OrderTransID    string `json:"order_trans_id"`
	OrderID         string `json:"order_id"`
	Status          string `json:"status"`
	ErrorCode       string `json:"error_code"`
	ErrorDesc       string `json:"error_desc"`
	ResponseMessage string `json:"response_message"`
}

// UpdateResponse for ..
type UpdateResponse struct {
	OrderTransID     string `json:"order_trans_id"`
	ErrorCode        string `json:"error_code"`
	ErrorDescription string `json:"error_description"`
}

// OrderRequest for mapping request of order
type OrderRequest struct {
	RequestDate    string            `json:"request_date"`
	RequestTransID string            `json:"request_trans_id"`
	TvsCustomerID  string            `json:"tvs_customer_id"`
	TvsReferenceID string            `json:"tvs_reference_id"`
	Ref1           string            `json:"ref1"`
	Ref2           string            `json:"ref2"`
	ByChannel      string            `json:"by_channel"`
	ByUser         string            `json:"by_user"`
	OrderType      string            `json:"order_type"`
	Payload        map[string]string `json:"payload"`
	//Payload        string `json:"payload"`
}

// OrderResponse for mapping response of service
type OrderResponse struct {
	OrderTransID     string            `json:"order_trans_id"`
	ErrorCode        string            `json:"error_code"`
	ErrorDescription string            `json:"error_description"`
	OrderType        string            `json:"order_type"`
	Activity         []Activity        `json:"activity,omitempty"`
	Payload          map[string]string `json:"payload,omitempty"`
	//Payload          string `json:"payload"`
}

// Activity for flow
type Activity struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Status             string `json:"status"`
	ErrorCode          string `json:"error_code"`
	ErrorDesc          string `json:"error_desc"`
	ResponseMessage    string `json:"response_message"`
	CreateDate         string `json:"create_date"`
	SentDate           string `json:"sent_date"`
	CompleteDate       string `json:"complete_date"`
	PreCondition       string `json:"pre_condition"`
	PreConditionResult string `json:"pre_condition_result"`
	RetryCondition     string `json:"retry_condition"`
}

// UpdatePayloadRequest for ...
type UpdatePayloadRequest struct {
	OrderTransID string            `json:"order_trans_id"`
	Payload      map[string]string `json:"payload"`
}

// UpdatePayloadResponse for ..
type UpdatePayloadResponse struct {
	OrderTransID     string `json:"order_trans_id"`
	ErrorCode        string `json:"error_code"`
	ErrorDescription string `json:"error_description"`
}

// UpdateOrderService to update transaction & transaction detail result
func UpdateOrderService(req UpdateRequest) UpdateResponse {
	var myReturn UpdateResponse

	// update order status
	var oErr int64
	dbPed := sms.New(cfg.dbPED)
	bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.ProcessUpdateService(:1,:2,:3,:4,:5,:6,:7,:8); end;",
		"UPDATE", req.OrderTransID, req.OrderID, req.Status, req.ErrorCode, req.ErrorDesc, req.ResponseMessage, sql.Out{Dest: &oErr})
	_ = bResult

	// submit order to queue by uuid
	go ProcessOrderService(req.OrderTransID)

	myReturn.OrderTransID = req.OrderTransID
	myReturn.ErrorCode = "0"
	myReturn.ErrorDescription = ""
	//myReturn.Activity = ""
	//myReturn.Payload = req.Payload

	return myReturn
}

// SubmitOrderService for ...
func SubmitOrderService(req OrderRequest) OrderResponse {
	var myReturn OrderResponse
	errCode := 0
	// Generate UUID
	orderUUID := pubModule.GenerateUUID()
	//log.Printf("UUID %s", orderUUID)

	// check mendatory field
	if req.RequestTransID == "" || req.TvsCustomerID == "" || req.OrderType == "" || req.Payload == nil {
		errCode = 400
	}

	if errCode == 0 {
		payloadStr, _ := json.Marshal(req.Payload)
		//log.Printf("payload %s", payloadStr)

		//#### Submit order to create flow
		var outTransID int64
		dbPed := sms.New(cfg.dbPED)
		bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.ProcessOrderService(:1,:2,:3,:4,:5,:6,:7,:8,:9,:10,:11); end;",
			req.TvsCustomerID, req.TvsReferenceID, req.Ref1, req.Ref2, req.ByChannel, req.ByUser, req.OrderType, string(payloadStr),
			req.RequestTransID, orderUUID, sql.Out{Dest: &outTransID})
		if bResult && outTransID > 0 {
			log.Printf("## Submit order to queue, Order Id=%d, UUID=%s", outTransID, orderUUID)
		} else {
			log.Printf("## Submit order fail.")
			errCode = 100
		}

		// submit order to queue by uuid
		go ProcessOrderService(orderUUID)
	}

	myReturn.OrderTransID = orderUUID
	myReturn.ErrorCode = strconv.Itoa(errCode)
	myReturn.ErrorDescription = viper.GetString("errorcode" + strconv.Itoa(errCode))
	myReturn.OrderType = req.OrderType
	//myReturn.Activity = ""
	//myReturn.Payload = req.Payload

	return myReturn
}

// ProcessOrderService for ..
func ProcessOrderService(orderTransID string) bool {
	bSuccess := true
	// submit next order to queue

	//log.Printf("Process Order")
	// ## Get next activity of this order
	act := getActivityFlow(orderTransID)
	if act != nil && len(act) > 0 {
		for i := 0; i < len(act); i++ {
			if act[i].Status == "N" {
				queueURL := cfg.queueurl

				if isAllowForPreConditionRule(orderTransID, act[i].Name) {
					// submit to queue
					log.Printf("## Route to queue [%s] url [%s]", act[i].Name, queueURL)

					q := SendQueue{queueURL, act[i].Name}
					ch := q.Connect()
					defer ch.Close()
					result := q.Send(ch, strconv.Itoa(act[i].ID), "", "application/json", orderTransID)
					var resultStatus string
					if result {
						resultStatus = "S"
					} else {
						resultStatus = "E"
						bSuccess = false
					}
					//log.Printf("Sent Queue status = %s", resultStatus)

					//Update Sent Result
					var oErr int64
					dbPed := sms.New(cfg.dbPED)
					bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.ProcessUpdateService(:1,:2,:3,:4,:5,:6,:7,:8); end;",
						"SEND", orderTransID, strconv.Itoa(act[i].ID), resultStatus, "", "", "", sql.Out{Dest: &oErr})
					_ = bResult

					return bSuccess
				}

				log.Printf("## Skip to queue [%s] url [%s], Pre-Condition not allow for this flow", act[i].Name, queueURL)

				//Update skip Result
				resultStatus := "Z"
				bSuccess = true

				var oErr int64
				dbPed := sms.New(cfg.dbPED)
				bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.ProcessUpdateService(:1,:2,:3,:4,:5,:6,:7,:8); end;",
					"SKIP", orderTransID, strconv.Itoa(act[i].ID), resultStatus, "0", "Pre-Condition not allow, this flow will be skip.", "", sql.Out{Dest: &oErr})
				_ = bResult
			}
		}
	}

	return bSuccess
}

// GetPayloadService for consumer get information of order
func GetPayloadService(req string) OrderResponse {
	var myReturn OrderResponse

	//log.Printf("## Process get order detail")
	//log.Printf("## Order Id = %s", req)

	//Get Order Info
	isError := 1
	orderType := ""
	var orderResult driver.Rows
	var payload map[string]string

	//log.Printf("Execute Store return cursor")
	dbPed := sms.New(cfg.dbPED)
	bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.GetPayloadData(:1,:2); end;", req, sql.Out{Dest: &orderResult})
	if bResult && orderResult != nil {
		values := make([]driver.Value, len(orderResult.Columns()))
		if orderResult.Next(values) == nil {
			orderType = values[0].(string)
			payloadStr := values[1].(string)

			if isJSON(payloadStr) {
				err := json.Unmarshal([]byte(payloadStr), &payload)
				if err != nil {
					//panic(err)
				}
			}
			isError = 0
		}
	}

	myReturn.OrderTransID = req
	myReturn.OrderType = orderType
	myReturn.Payload = payload
	myReturn.ErrorCode = strconv.Itoa(isError)
	myReturn.ErrorDescription = ""

	return myReturn
}

// GetOrderService for ..
func GetOrderService(req string) OrderResponse {
	var myReturn OrderResponse

	//log.Printf("## Process get order detail")
	//log.Printf("## Order Id = %s", req)

	//Get Order Info
	isError := 1
	orderType := ""
	var orderResult driver.Rows
	var payload map[string]string

	//log.Printf("Execute Store return cursor")
	dbPed := sms.New(cfg.dbPED)
	bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.GetPayloadData(:1,:2); end;", req, sql.Out{Dest: &orderResult})
	if bResult && orderResult != nil {
		values := make([]driver.Value, len(orderResult.Columns()))
		if orderResult.Next(values) == nil {
			orderType = values[0].(string)
			payloadStr := values[1].(string)

			if isJSON(payloadStr) {
				err := json.Unmarshal([]byte(payloadStr), &payload)
				if err != nil {
					//panic(err)
				}
			}

			isError = 0
		}
	}

	myReturn.OrderTransID = req
	myReturn.OrderType = orderType
	myReturn.Payload = payload
	myReturn.ErrorCode = strconv.Itoa(isError)
	myReturn.ErrorDescription = ""
	myReturn.Activity = getActivityFlow(req)

	return myReturn
}

func getActivityFlow(req string) []Activity {
	var myReturn []Activity

	var flowResult driver.Rows
	//log.Printf("Execute Store return cursor")
	dbPed := sms.New(cfg.dbPED)
	bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.GetActivityFlow(:1,:2); end;", req, sql.Out{Dest: &flowResult})
	if bResult && flowResult != nil {
		values := make([]driver.Value, len(flowResult.Columns()))
		for flowResult.Next(values) == nil {
			var act Activity
			act.ID, _ = strconv.Atoi(strconv.FormatInt(values[0].(int64), 10))
			act.Name = values[1].(string)
			act.Status = values[2].(string)
			act.ErrorCode = values[3].(string)
			act.ErrorDesc = values[4].(string)
			act.ResponseMessage = values[5].(string)
			act.CreateDate = values[6].(string)
			act.SentDate = values[7].(string)
			act.CompleteDate = values[8].(string)

			myReturn = append(myReturn, act)
		}
	}

	return myReturn
}

func isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// UpdatePayloadService for consumer get information of order
func UpdatePayloadService(req UpdatePayloadRequest) UpdatePayloadResponse {
	var myReturn UpdatePayloadResponse

	//log.Printf("## Process update payload")
	//log.Printf("## Order Id = %s", req.OrderTransID)

	//Get Order Info
	payloadStr, _ := json.Marshal(req.Payload)

	//log.Printf("Execute Store return cursor")
	var oErr int
	dbPed := sms.New(cfg.dbPED)
	bResult := dbPed.ExecuteStoreProcedure("begin PK_WFA_CORE.UpdatePayloadService(:1,:2,:3); end;", req.OrderTransID, string(payloadStr), sql.Out{Dest: &oErr})
	if bResult && oErr == 0 {
		// do something
	} else {
		oErr = 1
	}
	//log.Printf("resuit %t", bResult)

	myReturn.OrderTransID = req.OrderTransID
	myReturn.ErrorCode = strconv.Itoa(oErr)
	if oErr == 0 {
		myReturn.ErrorDescription = "Process Successful."
	} else {
		myReturn.ErrorDescription = "Cannot update payload."
	}

	return myReturn
}

func isAllowForPreConditionRule(orderTransID string, flowName string) bool {
	myReturn := false

	//### Skip
	//myReturn = false
	// TODO - Fix for flow = 'tvs_suspendsub' check payload suspendsubscriber = 'Y'
	if flowName == "tvs_suspendsub" {
		res := GetPayloadService(orderTransID)
		if res.ErrorCode == "0" {
			payload := res.Payload
			if payload["suspendsubscriber"] == "Y" {
				myReturn = true
			}
		}
	} else {
		myReturn = true
	}

	return myReturn
}
