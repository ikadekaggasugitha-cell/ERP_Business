package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestSalesQuotationsConvertAndReturn(t *testing.T) {
	store := newMemSalesStore()
	router := NewRouter(store, nil)
	orgID := uuid.NewString()
	userID := uuid.NewString()
	custID := uuid.NewString()
	itemID := uuid.NewString()

	call := func(method, path string, body any, idem, ifMatch string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-Permissions", "sales:read,sales:write")
		req.Header.Set("Content-Type", "application/json")
		if idem != "" {
			req.Header.Set("Idempotency-Key", idem)
		}
		if ifMatch != "" {
			req.Header.Set("If-Match", ifMatch)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	// 1. Create Sales Quotation
	qBody := map[string]any{
		"quotationNo":  "SQ-2026-001",
		"customerId":   custID,
		"currencyCode": "IDR",
		"lines":        []map[string]any{{"lineNo": 1, "itemId": itemID, "orderQty": "5.0000", "unitPrice": "200000.0000"}},
	}
	rec := call(http.MethodPost, "/sales-quotations", qBody, "sq-1", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("create quotation expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var sq salesQuotationDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &sq)
	if sq.TotalAmount != "1000000.0000" || sq.Status != "Open" {
		t.Fatalf("unexpected quotation: %+v", sq)
	}

	// 2. List Sales Quotations
	recList := call(http.MethodGet, "/sales-quotations", nil, "", "")
	if recList.Code != http.StatusOK {
		t.Fatalf("list quotations expected 200, got %d", recList.Code)
	}

	// 3. Convert SQ -> SO
	recConv := call(http.MethodPost, "/sales-quotations/"+sq.ID+"/convert-to-order", map[string]any{"soNo": "SO-FROM-SQ1"}, "sq-conv-1", "")
	if recConv.Code != http.StatusCreated {
		t.Fatalf("convert quotation expected 201, got %d (%s)", recConv.Code, recConv.Body.String())
	}
	var convResp struct {
		Quotation  salesQuotationDTO `json:"quotation"`
		SalesOrder soDetailResp      `json:"salesOrder"`
	}
	_ = json.Unmarshal(recConv.Body.Bytes(), &convResp)
	if convResp.Quotation.Status != "Converted" || convResp.SalesOrder.SoNo != "SO-FROM-SQ1" {
		t.Fatalf("unexpected convert result: %+v", convResp)
	}

	// 4. Confirm SO, then Return 2 units (Credit Memo = 400,000.0000)
	recConf := call(http.MethodPost, "/sales-orders/"+convResp.SalesOrder.ID+"/confirm", nil, "", `"1"`)
	if recConf.Code != http.StatusOK {
		t.Fatalf("confirm SO expected 200, got %d (%s)", recConf.Code, recConf.Body.String())
	}
	recRet := call(http.MethodPost, "/sales-orders/"+convResp.SalesOrder.ID+"/return", map[string]any{
		"reason": "damaged in transit",
		"lines":  []map[string]any{{"itemId": itemID, "returnQty": "2.0000"}},
	}, "ret-1", "")
	if recRet.Code != http.StatusCreated {
		t.Fatalf("return SO expected 201, got %d (%s)", recRet.Code, recRet.Body.String())
	}
	var retResp map[string]any
	_ = json.Unmarshal(recRet.Body.Bytes(), &retResp)
	if retResp["creditMemoAmount"] != "400000.0000" || retResp["creditMemoStatus"] != "Pending" {
		t.Fatalf("unexpected return response: %+v", retResp)
	}

	// Exceeding return qty must return 422
	recBad := call(http.MethodPost, "/sales-orders/"+convResp.SalesOrder.ID+"/return", map[string]any{
		"reason": "too many",
		"lines":  []map[string]any{{"itemId": itemID, "returnQty": "9.0000"}},
	}, "ret-bad", "")
	if recBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for excessive return qty, got %d", recBad.Code)
	}
}
