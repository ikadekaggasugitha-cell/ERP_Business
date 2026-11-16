package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/calc"
	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/payrolldb"
	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type memPayrollStore struct {
	runs map[uuid.UUID]repository.PayrollRunDetail
}

func newMemPayrollStore() *memPayrollStore {
	return &memPayrollStore{runs: make(map[uuid.UUID]repository.PayrollRunDetail)}
}

func (m *memPayrollStore) CreateRun(_ context.Context, _ []uuid.UUID, orgID uuid.UUID, runNo string, month, year int32, corrID, _ uuid.UUID) (repository.PayrollRunDetail, error) {
	id := uuid.New()
	d := repository.PayrollRunDetail{
		ID: id, OrganizationID: orgID, RunNo: runNo, PeriodMonth: month, PeriodYear: year,
		Status: "Draft", CorrelationID: corrID, RowVersion: 1,
	}
	m.runs[id] = d
	return d, nil
}

func (m *memPayrollStore) GetByID(_ context.Context, _ []uuid.UUID, id uuid.UUID) (repository.PayrollRunDetail, error) {
	d, ok := m.runs[id]
	if !ok {
		return repository.PayrollRunDetail{}, httpx.NotFound("payroll run tidak ditemukan")
	}
	return d, nil
}

func (m *memPayrollStore) ListWithCount(_ context.Context, _ []uuid.UUID, status *string, _, _ int32) ([]payrolldb.ListPayrollRunsRow, int64, error) {
	var out []payrolldb.ListPayrollRunsRow
	for _, d := range m.runs {
		if status != nil && d.Status != *status {
			continue
		}
		var ref *string
		if d.GLJournalRef != "" {
			s := d.GLJournalRef
			ref = &s
		}
		out = append(out, payrolldb.ListPayrollRunsRow{
			ID: d.ID, OrganizationID: d.OrganizationID, RunNo: d.RunNo, PeriodMonth: d.PeriodMonth,
			PeriodYear: d.PeriodYear, Status: d.Status, GlJournalRef: ref, RowVersion: d.RowVersion,
		})
	}
	return out, int64(len(out)), nil
}

func (m *memPayrollStore) CalculateAndEmit(_ context.Context, _ []uuid.UUID, id uuid.UUID, version int32, _ *big.Rat, employees []calc.EmployeeInput, _ uuid.UUID) (repository.PayrollRunDetail, error) {
	d, ok := m.runs[id]
	if !ok {
		return repository.PayrollRunDetail{}, httpx.NotFound("payroll run tidak ditemukan")
	}
	if d.Status != "Draft" || d.RowVersion != version {
		return repository.PayrollRunDetail{}, httpx.Conflict("VERSION_CONFLICT", "conflict")
	}
	d.Results = nil
	for _, e := range employees {
		comp, err := calc.ComputeEmployee(e)
		if err != nil {
			return repository.PayrollRunDetail{}, httpx.Unprocessable("CALCULATION_ERROR", err.Error())
		}
		d.Results = append(d.Results, repository.PayrollResultDTO{
			ID:              uuid.New(),
			EmployeeID:      e.EmployeeID,
			GrossAmount:     platform.FormatNumeric(comp.Gross, 4),
			DeductionAmount: platform.FormatNumeric(comp.Deduction, 4),
			TaxAmount:       platform.FormatNumeric(comp.TaxPPh21, 4),
			NetAmount:       platform.FormatNumeric(comp.Net, 4),
			RowVersion:      1,
		})
	}
	d.Status = "Calculated"
	d.RowVersion++
	m.runs[id] = d
	return d, nil
}

func (m *memPayrollStore) ApproveAndEmit(_ context.Context, _ []uuid.UUID, id uuid.UUID, version int32, _, _ uuid.UUID) (repository.PayrollRunDetail, error) {
	d, ok := m.runs[id]
	if !ok {
		return repository.PayrollRunDetail{}, httpx.NotFound("payroll run tidak ditemukan")
	}
	if d.Status != "Calculated" || d.RowVersion != version {
		return repository.PayrollRunDetail{}, httpx.Conflict("INVALID_STATE", "conflict")
	}
	d.Status = "Approved"
	d.RowVersion++
	m.runs[id] = d
	return d, nil
}

func (m *memPayrollStore) PostAndEmit(_ context.Context, _ []uuid.UUID, id uuid.UUID, version int32, ref string, _, _ uuid.UUID) (repository.PayrollRunDetail, error) {
	d, ok := m.runs[id]
	if !ok {
		return repository.PayrollRunDetail{}, httpx.NotFound("payroll run tidak ditemukan")
	}
	if (d.Status != "Calculated" && d.Status != "Approved") || d.RowVersion != version {
		return repository.PayrollRunDetail{}, httpx.Conflict("INVALID_STATE", "conflict")
	}
	d.Status = "Posted"
	if ref == "" {
		ref = "AUTO-" + d.RunNo
	}
	d.GLJournalRef = ref
	d.RowVersion++
	m.runs[id] = d
	return d, nil
}

func (m *memPayrollStore) DeleteDraft(_ context.Context, _ []uuid.UUID, id uuid.UUID, version int32, _ uuid.UUID) error {
	d, ok := m.runs[id]
	if !ok {
		return httpx.NotFound("payroll run tidak ditemukan")
	}
	if d.Status != "Draft" || d.RowVersion != version {
		return httpx.Conflict("INVALID_STATE", "conflict")
	}
	delete(m.runs, id)
	return nil
}

func doPayReq(t *testing.T, h http.Handler, method, path, body, perm, ifMatch string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("X-Organization-Id", uuid.NewString())
	req.Header.Set("X-User-Id", uuid.NewString())
	if perm != "" {
		req.Header.Set("X-Permissions", perm)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestPayrollRunLifecycleCalculationAndRBAC(t *testing.T) {
	store := newMemPayrollStore()
	router := NewRouter(store, nil)

	// 1. RBAC reject write
	rec := doPayReq(t, router, http.MethodPost, "/payroll-runs", `{"runNo":"RUN-1","periodMonth":3,"periodYear":2026}`, "payroll:read", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	// 2. Create Draft Run -> 201
	rec = doPayReq(t, router, http.MethodPost, "/payroll-runs", `{"runNo":"RUN-01","periodMonth":3,"periodYear":2026}`, "payroll:write", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	var created repository.PayrollRunDetail
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// 3. Calculate batched employee -> 200
	empID := uuid.NewString()
	calcBody := `{"employees":[{"employeeId":"` + empID + `","basicSalary":"15000000.0000","ptkp":"TK/0"}]}`
	rec = doPayReq(t, router, http.MethodPost, "/payroll-runs/"+created.ID.String()+"/calculate", calcBody, "payroll:write", `"1"`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 calc, got %d %s", rec.Code, rec.Body.String())
	}
	var calced repository.PayrollRunDetail
	_ = json.Unmarshal(rec.Body.Bytes(), &calced)
	if calced.Status != "Calculated" || len(calced.Results) != 1 || calced.Results[0].TaxAmount != "939925.0000" {
		t.Fatalf("unexpected calculated run: %+v", calced)
	}

	// 4. Approve -> Post
	rec = doPayReq(t, router, http.MethodPost, "/payroll-runs/"+created.ID.String()+"/approve", "", "payroll:write", `"2"`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 approve, got %d %s", rec.Code, rec.Body.String())
	}
	rec = doPayReq(t, router, http.MethodPost, "/payroll-runs/"+created.ID.String()+"/post", `{"glJournalRef":"JV-PAY-01"}`, "payroll:write", `"3"`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 post, got %d %s", rec.Code, rec.Body.String())
	}
	var posted repository.PayrollRunDetail
	_ = json.Unmarshal(rec.Body.Bytes(), &posted)
	if posted.Status != "Posted" || posted.GLJournalRef != "JV-PAY-01" {
		t.Fatalf("unexpected posted run: %+v", posted)
	}
}

