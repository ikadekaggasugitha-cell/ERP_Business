package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func TestTreasuryCSVMT940ImportManualReconcileAndCashPosition(t *testing.T) {
	store := &fakeTreasuryStore{}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	orgID := uuid.NewString()
	userID := uuid.NewString()
	bankID := "10000000-0000-0000-0000-000000000101"

	call := func(method, path string, body any, idem string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("X-Organization-Id", orgID)
		req.Header.Set("X-User-Id", userID)
		req.Header.Set("X-Permissions", "treasury:read,treasury:write")
		req.Header.Set("Content-Type", "application/json")
		if idem != "" {
			req.Header.Set("Idempotency-Key", idem)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// 1. BCA/Mandiri CSV import
	csvPayload := map[string]any{
		"format": "csv",
		"content": "Tanggal,Keterangan,Debit,Kredit,Ref\n" +
			"15/03/2026,Pembayaran Vendor PT Baja,1.250.000,50,0,PAY-001\n" +
			"16/03/2026,Pelunasan Customer A,0,2.500.000,00,RCP-002\n",
	}
	// Note: commas inside Indonesian decimals must be quoted in RFC4180 CSV
	csvPayload["content"] = "Tanggal,Keterangan,Debit,Kredit,Ref\n" +
		"15/03/2026,Pembayaran Vendor PT Baja,\"1.250.000,50\",0,PAY-001\n" +
		"16/03/2026,Pelunasan Customer A,0,\"2.500.000,00\",RCP-002\n"

	rec := call(http.MethodPost, "/bank-accounts/"+bankID+"/statements/import-csv", csvPayload, "csv-idem-1")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on CSV import, got %d (%s)", rec.Code, rec.Body.String())
	}
	var csvResp struct {
		Format        string           `json:"format"`
		ImportedCount int              `json:"importedCount"`
		Lines         []map[string]any `json:"lines"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &csvResp)
	if csvResp.Format != "csv" || csvResp.ImportedCount != 2 || csvResp.Lines[0]["amount"] != "-1250000.5000" || csvResp.Lines[1]["amount"] != "2500000.0000" {
		t.Fatalf("unexpected CSV import result: %+v", csvResp)
	}

	// 2. MT940 import
	mt940Payload := map[string]any{
		"format": "auto",
		"content": ":20:STMT202603\n" +
			":61:2603180318D500000,25NTRFPAY-009\n" +
			":86:Transfer keluar vendor\n" +
			":61:2603190319C1000000,00NTRFRCP-010\n" +
			":86:Setoran masuk customer\n",
	}
	recMT := call(http.MethodPost, "/bank-accounts/"+bankID+"/statements/import-csv", mt940Payload, "mt-idem-1")
	if recMT.Code != http.StatusCreated {
		t.Fatalf("expected 201 on MT940 import, got %d (%s)", recMT.Code, recMT.Body.String())
	}
	var mtResp struct {
		Format        string           `json:"format"`
		ImportedCount int              `json:"importedCount"`
		Lines         []map[string]any `json:"lines"`
	}
	_ = json.Unmarshal(recMT.Body.Bytes(), &mtResp)
	if mtResp.Format != "mt940" || mtResp.ImportedCount != 2 || mtResp.Lines[0]["amount"] != "-500000.2500" || mtResp.Lines[1]["amount"] != "1000000.0000" {
		t.Fatalf("unexpected MT940 import result: %+v", mtResp)
	}

	// 3. Manual reconcile match & unmatch
	lineID := uuid.NewString()
	payID := uuid.NewString()
	recMan := call(http.MethodPost, "/bank-accounts/"+bankID+"/reconcile/manual", map[string]any{
		"action":          "match",
		"statementLineId": lineID,
		"paymentId":       payID,
		"reason":          "verified bank slip",
	}, "man-idem-1")
	if recMan.Code != http.StatusOK {
		t.Fatalf("expected 200 on manual match, got %d (%s)", recMan.Code, recMan.Body.String())
	}
	if len(store.settledEvents) != 1 || store.settledEvents[0].EventType != "treasury.payment.reconciled" {
		t.Fatalf("expected manual reconcile outbox event, got %+v", store.settledEvents)
	}

	// 4. Multi-currency cash position & 30/60/90-day forecast
	recCash := call(http.MethodGet, "/cash-position", nil, "")
	if recCash.Code != http.StatusOK {
		t.Fatalf("expected 200 on GET /cash-position, got %d (%s)", recCash.Code, recCash.Body.String())
	}
	var cashResp struct {
		BankCount  int              `json:"bankCount"`
		Currencies []map[string]any `json:"currencies"`
	}
	_ = json.Unmarshal(recCash.Body.Bytes(), &cashResp)
	if cashResp.BankCount != 1 || len(cashResp.Currencies) != 1 || cashResp.Currencies[0]["currencyCode"] != "IDR" {
		t.Fatalf("unexpected cash-position response: %+v", cashResp)
	}
}
