package httpapi

type poLineReq struct {
	LineNo    int32  `json:"lineNo"`
	ItemID    string `json:"itemId"`
	OrderQty  string `json:"orderQty"`
	UnitPrice string `json:"unitPrice"`
}

type poUpsertReq struct {
	PoNo                  string      `json:"poNo"`
	VendorID              string      `json:"vendorId"`
	PurchaseRequisitionID string      `json:"purchaseRequisitionId,omitempty"`
	OrderDate             string      `json:"orderDate"`
	CurrencyCode          string      `json:"currencyCode"`
	TotalAmount           string      `json:"totalAmount"`
	Lines                 []poLineReq `json:"lines"`
}

type receiveReq struct {
	GrNo        string `json:"grNo,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
	ReceiptDate string `json:"receiptDate,omitempty"`
}

type poLineDTO struct {
	ID          string `json:"id,omitempty"`
	LineNo      int32  `json:"lineNo"`
	ItemID      string `json:"itemId"`
	OrderQty    string `json:"orderQty"`
	ReceivedQty string `json:"receivedQty"`
	UnitPrice   string `json:"unitPrice"`
}

type poDTO struct {
	ID                    string      `json:"id"`
	PoNo                  string      `json:"poNo,omitempty"`
	VendorID              string      `json:"vendorId,omitempty"`
	PurchaseRequisitionID string      `json:"purchaseRequisitionId,omitempty"`
	OrderDate             string      `json:"orderDate,omitempty"`
	CurrencyCode          string      `json:"currencyCode,omitempty"`
	TotalAmount           string      `json:"totalAmount,omitempty"`
	Status                string      `json:"status"`
	RowVersion            int32       `json:"rowVersion"`
	GrID                  string      `json:"grId,omitempty"`
	GrNo                  string      `json:"grNo,omitempty"`
	Lines                 []poLineDTO `json:"lines,omitempty"`
}

type grLineDTO struct {
	ID          string `json:"id"`
	PoLineID    string `json:"poLineId"`
	LineNo      int32  `json:"lineNo"`
	ReceivedQty string `json:"receivedQty"`
}

type grDTO struct {
	ID              string      `json:"id"`
	GrNo            string      `json:"grNo"`
	PurchaseOrderID string      `json:"purchaseOrderId"`
	WarehouseID     string      `json:"warehouseId"`
	ReceiptDate     string      `json:"receiptDate"`
	Status          string      `json:"status"`
	RowVersion      int32       `json:"rowVersion"`
	Lines           []grLineDTO `json:"lines,omitempty"`
}

type prLineReq struct {
	LineNo     int32  `json:"lineNo"`
	ItemID     string `json:"itemId"`
	RequestQty string `json:"requestQty"`
	NeededDate string `json:"neededDate,omitempty"`
}

type prCreateReq struct {
	PrNo        string      `json:"prNo"`
	RequesterID string      `json:"requesterId,omitempty"`
	WorkflowRef string      `json:"workflowRef,omitempty"`
	NeededDate  string      `json:"neededDate,omitempty"`
	Lines       []prLineReq `json:"lines"`
}

type prLineDTO struct {
	ID         string `json:"id"`
	LineNo     int32  `json:"lineNo"`
	ItemID     string `json:"itemId"`
	RequestQty string `json:"requestQty"`
	NeededDate string `json:"neededDate,omitempty"`
}

type prDTO struct {
	ID          string      `json:"id"`
	PrNo        string      `json:"prNo,omitempty"`
	RequesterID string      `json:"requesterId,omitempty"`
	WorkflowRef string      `json:"workflowRef,omitempty"`
	NeededDate  string      `json:"neededDate,omitempty"`
	Status      string      `json:"status"`
	RowVersion  int32       `json:"rowVersion"`
	Lines       []prLineDTO `json:"lines,omitempty"`
}

