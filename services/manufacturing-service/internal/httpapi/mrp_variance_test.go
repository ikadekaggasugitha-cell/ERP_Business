package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func TestManufacturingMRPPlanAndWorkOrderVariance(t *testing.T) {
	store := newFakeMfgStore()
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	handler := httpx.Middleware(mux)

	orgID := uuid.NewString()
	userID := uuid.NewString()
	prodID := uuid.NewString()
	compID := uuid.NewString()
	rwPerms := "manufacturing:read,manufacturing:write"

	// 1. Create BOM: 2.5 units of compID per finished unit
	bomBody := map[string]any{
		"code": "BOM-MRP", "productItemId": prodID,
		"lines": []map[string]any{{"lineNo": 1, "componentItemId": compID, "quantity": "2.5000", "uom": "KG"}},
	}
	rec := doMfgReq(t, handler, http.MethodPost, "/bill-of-materials", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "bom-mrp"}, bomBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create BOM expected 201, got %d", rec.Code)
	}
	var bom bomDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &bom)

	// 2. MRP Plan for 10 units -> requires 25.0000 KG; available 15.0000 -> netShortage 10.0000
	mrpBody := map[string]any{
		"bomId":          bom.ID,
		"targetQty":      "10.0000",
		"availableStock": map[string]string{compID: "15.0000"},
	}
	recMRP := doMfgReq(t, handler, http.MethodPost, "/work-orders/mrp-plan", orgID, userID, rwPerms, nil, mrpBody)
	if recMRP.Code != http.StatusOK {
		t.Fatalf("mrp-plan expected 200, got %d (%s)", recMRP.Code, recMRP.Body.String())
	}
	var mrpResp struct {
		ShortageItemCount int              `json:"shortageItemCount"`
		CanFulfillFromInv bool             `json:"canFulfillFromInv"`
		Requirements      []map[string]any `json:"requirements"`
	}
	_ = json.Unmarshal(recMRP.Body.Bytes(), &mrpResp)
	if mrpResp.ShortageItemCount != 1 || mrpResp.CanFulfillFromInv || mrpResp.Requirements[0]["netShortageQty"] != "10.0000" {
		t.Fatalf("unexpected MRP plan result: %+v", mrpResp)
	}

	// 3. Create WO (planned=10) -> Release -> Complete with produced=8 -> Variance shows scrap=2, yield=80.00%
	recWO := doMfgReq(t, handler, http.MethodPost, "/work-orders", orgID, userID, rwPerms, map[string]string{"Idempotency-Key": "wo-var"}, map[string]any{
		"woNo": "WO-VAR-1", "productItemId": prodID, "bomId": bom.ID, "plannedQty": "10.0000",
	})
	var wo woDTO
	_ = json.Unmarshal(recWO.Body.Bytes(), &wo)
	_ = doMfgReq(t, handler, http.MethodPost, "/work-orders/"+wo.ID+"/release", orgID, userID, rwPerms, map[string]string{"If-Match": "1"}, nil)
	_ = doMfgReq(t, handler, http.MethodPost, "/work-orders/"+wo.ID+"/complete", orgID, userID, rwPerms, map[string]string{"If-Match": "2"}, map[string]any{"producedQty": "8.0000"})

	recVar := doMfgReq(t, handler, http.MethodGet, "/work-orders/"+wo.ID+"/variance?standardUnitCost=1000.0000", orgID, userID, "manufacturing:read", nil, nil)
	if recVar.Code != http.StatusOK {
		t.Fatalf("variance expected 200, got %d (%s)", recVar.Code, recVar.Body.String())
	}
	var varResp map[string]any
	_ = json.Unmarshal(recVar.Body.Bytes(), &varResp)
	if varResp["scrapQty"] != "2.0000" || varResp["yieldPct"] != "80.00" || varResp["costVariance"] != "2000.0000" {
		t.Fatalf("unexpected variance response: %+v", varResp)
	}
}
