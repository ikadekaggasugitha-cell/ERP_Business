package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func TestProcurementReturnToVendor(t *testing.T) {
	store := newFakePOStore()
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	handler := httpx.Middleware(mux)

	orgID := uuid.NewString()
	userID := uuid.NewString()
	approverID := uuid.NewString()
	vendorID := uuid.NewString()
	itemID := uuid.NewString()
	rwPerms := "procurement:read,procurement:write"
	aprPerms := "procurement:read,procurement:approve"

	poBody := map[string]any{
		"poNo": "PO-RTV-01", "vendorId": vendorID, "orderDate": "2026-03-15", "currencyCode": "IDR", "totalAmount": "5000.0000",
		"lines": []map[string]any{{"lineNo": 1, "itemId": itemID, "orderQty": "10.0000", "unitPrice": "500.0000"}},
	}
	rec := doPOReq(t, handler, http.MethodPost, "/purchase-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "k-rtv-po"}, poBody)
	var po poDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &po)

	_ = doPOReq(t, handler, http.MethodPost, "/purchase-orders/"+po.ID+"/approve", orgID, approverID, aprPerms, map[string]string{"If-Match": "1"}, nil)
	recGR := doPOReq(t, handler, http.MethodPost, "/purchase-orders/"+po.ID+"/receive", orgID, userID, rwPerms, map[string]string{"If-Match": "2"}, nil)
	var grPO poDTO
	_ = json.Unmarshal(recGR.Body.Bytes(), &grPO)

	// Valid RTV: return 3 out of 10 received
	rtvBody := map[string]any{
		"reason": "defective batch",
		"lines":  []map[string]any{{"itemId": itemID, "returnQty": "3.0000"}},
	}
	recRTV := doPOReq(t, handler, http.MethodPost, "/goods-receipts/"+grPO.GrID+"/return-to-vendor", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "rtv-1"}, rtvBody)
	if recRTV.Code != http.StatusCreated {
		t.Fatalf("RTV expected 201, got %d (%s)", recRTV.Code, recRTV.Body.String())
	}
	var rtvResp map[string]any
	_ = json.Unmarshal(recRTV.Body.Bytes(), &rtvResp)
	if rtvResp["totalReturnQty"] != "3.0000" || rtvResp["status"] != "Returned" {
		t.Fatalf("unexpected RTV response: %+v", rtvResp)
	}

	// Excessive RTV (> 10 received) -> 422
	rtvBad := map[string]any{
		"reason": "excessive",
		"lines":  []map[string]any{{"itemId": itemID, "returnQty": "15.0000"}},
	}
	recBad := doPOReq(t, handler, http.MethodPost, "/goods-receipts/"+grPO.GrID+"/return-to-vendor", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "rtv-bad"}, rtvBad)
	if recBad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for excessive RTV qty, got %d (%s)", recBad.Code, recBad.Body.String())
	}
}
