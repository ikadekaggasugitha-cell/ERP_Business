package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func TestProcurementThreeWayMatch(t *testing.T) {
	store := newFakePOStore()
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	handler := httpx.Middleware(mux)

	orgID := uuid.NewString()
	userID := uuid.NewString()
	vendorID := uuid.NewString()
	itemID := uuid.NewString()
	rwPerms := "procurement:read,procurement:write"

	// Create PO: 10 units @ 1000.0000 = 10,000.0000
	poBody := map[string]any{
		"poNo": "PO-3WM-01", "vendorId": vendorID, "orderDate": "2026-03-15", "currencyCode": "IDR", "totalAmount": "10000.0000",
		"lines": []map[string]any{{"lineNo": 1, "itemId": itemID, "orderQty": "10.0000", "unitPrice": "1000.0000"}},
	}
	rec := doPOReq(t, handler, http.MethodPost, "/purchase-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-3wm"}, poBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create PO expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var created poDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// 1. WithinTolerance: GR=10, Invoice=10 @ 1020 (+2% price within 5% tolerance)
	matchBody := map[string]any{
		"invoiceNo":         "INV-V-100",
		"qtyTolerancePct":   "1.00",
		"priceTolerancePct": "5.00",
		"grLines":           []map[string]any{{"itemId": itemID, "receivedQty": "10.0000"}},
		"invoiceLines":      []map[string]any{{"itemId": itemID, "invoicedQty": "10.0000", "unitPrice": "1020.0000"}},
	}
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-orders/"+created.ID+"/three-way-match", orgID, userID, rwPerms, nil, matchBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("three-way-match expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var res threeWayMatchResp
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !res.Matched || res.Status != "WithinTolerance" || res.TotalVariance != "200.0000" {
		t.Fatalf("unexpected within-tolerance result: %+v", res)
	}

	// 2. VarianceExceeded: Invoice qty 12 vs GR 10 (+20% qty exceeds 1% tolerance)
	matchBody["invoiceLines"] = []map[string]any{{"itemId": itemID, "invoicedQty": "12.0000", "unitPrice": "1000.0000"}}
	rec = doPOReq(t, handler, http.MethodPost, "/purchase-orders/"+created.ID+"/three-way-match", orgID, userID, rwPerms, nil, matchBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("three-way-match expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var resFail threeWayMatchResp
	_ = json.Unmarshal(rec.Body.Bytes(), &resFail)
	if resFail.Matched || resFail.Status != "VarianceExceeded" || resFail.Lines[0].QtyVariance != "2.0000" {
		t.Fatalf("unexpected variance-exceeded result: %+v", resFail)
	}
}
