package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"log"
	"strconv"

	pubModule "github.com/patomp3/wfacore/module"
)

/*type PayloadInfo struct {
	Key string `json:"key"`
	Value string `json:"value"`
}*/

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
	Activity         []Activity        `json:"activity"`
	Payload          map[string]string `json:"payload"`
	//Payload          string `json:"payload"`
}

// Activity for flow
type Activity struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	ErrorCode       string `json:"error_code"`
	ErrorDesc       string `json:"error_desc"`
	ResponseMessage string `json:"response_message"`
	CreateDate      string `json:"create_date"`
	SentDate        string `json:"sent_date"`
	CompleteDate    string `json:"complete_date"`
}

// UpdateOrderService for ..
func UpdateOrderService(req UpdateRequest) UpdateResponse {
	var myReturn UpdateResponse

	// update order status
	var oErr int64
	bResult := ExecuteStoreProcedure("QED", "begin PK_WFA_CORE.ProcessUpdateService(:1,:2,:3,:4,:5,:6,:7,:8); end;",
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

	// Generate UUID
	orderUUID := pubModule.GenerateUUID()
	log.Printf("UUID %s", orderUUID)

	payloadStr, _ := json.Marshal(req.Payload)
	//log.Printf("payload %s", payloadStr)

	//#### Submit order to create flow
	var outTransID int64
	bResult := ExecuteStoreProcedure("QED", "begin PK_WFA_CORE.ProcessOrderService(:1,:2,:3,:4,:5,:6,:7,:8,:9,:10,:11); end;",
		req.TvsCustomerID, req.TvsReferenceID, req.Ref1, req.Ref2, req.ByChannel, req.ByUser, req.OrderType, string(payloadStr),
		req.RequestTransID, orderUUID, sql.Out{Dest: &outTransID})
	if bResult {
		log.Printf("Submit to Queue, Order Id=%d, UUID=%s", outTransID, orderUUID)
	} else {
		log.Printf("Fail to submit order.")
	}

	// submit order to queue by uuid
	go ProcessOrderService(orderUUID)

	myReturn.OrderTransID = orderUUID
	myReturn.ErrorCode = "0"
	myReturn.ErrorDescription = ""
	//myReturn.Activity = ""
	//myReturn.Payload = req.Payload

	return myReturn
}

// ProcessOrderService for ..
func ProcessOrderService(orderTransID string) bool {
	var bSuccess bool
	// submit next order to queue

	//log.Printf("Process Order")
	// ## Get next activity of this order
	act := getActivityFlow(orderTransID)
	if act != nil && len(act) > 0 {
		for i := 0; i < len(act); i++ {
			if act[i].Status == "N" {
				// submit to queue
				queueURL := cfg.queueurl
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
				}
				//log.Printf("Sent Queue status = %s", resultStatus)

				//Update Sent Result
				var oErr int64
				bResult := ExecuteStoreProcedure("QED", "begin PK_WFA_CORE.ProcessUpdateService(:1,:2,:3,:4,:5,:6,:7,:8); end;",
					"SEND", orderTransID, strconv.Itoa(act[i].ID), resultStatus, "", "", "", sql.Out{Dest: &oErr})
				_ = bResult

				return bSuccess
			}
		}
	}
	//log.Printf("End Order")

	return bSuccess
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
	bResult := ExecuteStoreProcedure("QED", "begin PK_WFA_CORE.GetPayloadData(:1,:2); end;", req, sql.Out{Dest: &orderResult})
	if bResult && orderResult != nil {
		values := make([]driver.Value, len(orderResult.Columns()))
		if orderResult.Next(values) == nil {
			orderType = values[0].(string)
			payloadStr := values[1].(string)

			err := json.Unmarshal([]byte(payloadStr), &payload)
			if err != nil {
				//panic(err)
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
	bResult := ExecuteStoreProcedure("QED", "begin PK_WFA_CORE.GetActivityFlow(:1,:2); end;", req, sql.Out{Dest: &flowResult})
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
