package httpapi

type bomLineReq struct {
	LineNo          int32  `json:"lineNo"`
	ComponentItemID string `json:"componentItemId"`
	Quantity        string `json:"quantity"`
	Uom             string `json:"uom"`
}

type bomReq struct {
	Code          string       `json:"code"`
	ProductItemID string       `json:"productItemId"`
	Version       int32        `json:"version,omitempty"`
	Lines         []bomLineReq `json:"lines,omitempty"`
}

type bomLineDTO struct {
	ID              string `json:"id"`
	LineNo          int32  `json:"lineNo"`
	ComponentItemID string `json:"componentItemId"`
	Quantity        string `json:"quantity"`
	Uom             string `json:"uom"`
}

type bomDTO struct {
	ID            string       `json:"id"`
	Code          string       `json:"code"`
	ProductItemID string       `json:"productItemId"`
	Version       int32        `json:"version"`
	IsActive      bool         `json:"isActive"`
	RowVersion    int32        `json:"rowVersion"`
	Lines         []bomLineDTO `json:"lines,omitempty"`
}

type woUpsertReq struct {
	WoNo          string `json:"woNo"`
	ProductItemID string `json:"productItemId"`
	BomID         string `json:"bomId"`
	PlannedQty    string `json:"plannedQty"`
	SalesOrderRef string `json:"salesOrderRef,omitempty"`
}

type completeReq struct {
	ProducedQty string `json:"producedQty,omitempty"`
	UnitCost    string `json:"unitCost,omitempty"`
	WarehouseID string `json:"warehouseId,omitempty"`
}

type woDTO struct {
	ID            string `json:"id"`
	WoNo          string `json:"woNo,omitempty"`
	ProductItemID string `json:"productItemId,omitempty"`
	BomID         string `json:"bomId,omitempty"`
	PlannedQty    string `json:"plannedQty,omitempty"`
	ProducedQty   string `json:"producedQty,omitempty"`
	SalesOrderRef string `json:"salesOrderRef,omitempty"`
	Status        string `json:"status"`
	RowVersion    int32  `json:"rowVersion"`
}

