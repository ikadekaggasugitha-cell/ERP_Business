package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func TestTreasuryExportDisbursementCSV(t *testing.T) {
	store := &fakeTreasuryStore{}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	req := httptest.NewRequest(http.MethodGet, "/payments/export/disbursement?format=BIFAST", nil)
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	req.Header.Set("X-Permissions", "treasury:read")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on disbursement export, got %d (%s)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected text/csv content-type, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "transfer_channel,payment_no") || !strings.Contains(body, "BIFAST,PAY-001") {
		t.Fatalf("unexpected disbursement CSV: %s", body)
	}

	// Formula injection neutralization check
	if got := httpx.SanitizeCSVCell("=SUM(A1:A2)"); got != "'=SUM(A1:A2)" {
		t.Fatalf("expected formula neutralization, got %q", got)
	}

	// Invalid format -> 400
	reqBad := httptest.NewRequest(http.MethodGet, "/payments/export/disbursement?format=WIRE", nil)
	reqBad.Header.Set("X-Organization-Id", uuid.NewString())
	reqBad.Header.Set("X-User-Id", uuid.NewString())
	reqBad.Header.Set("X-Permissions", "treasury:read")
	recBad := httptest.NewRecorder()
	h.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid format, got %d", recBad.Code)
	}
}
