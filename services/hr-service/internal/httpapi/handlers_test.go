package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/hrdb"
	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type memHRStore struct {
	employees map[uuid.UUID]repository.EmployeeDTO
	byHash    map[string]uuid.UUID
	key       []byte
}

func newMemHRStore() *memHRStore {
	key, _ := platform.LoadPIIKey()
	return &memHRStore{
		employees: make(map[uuid.UUID]repository.EmployeeDTO),
		byHash:    make(map[string]uuid.UUID),
		key:       key,
	}
}

func (m *memHRStore) CreateAndEmit(_ context.Context, _ []uuid.UUID, in repository.CreateEmployeeInput) (repository.EmployeeDTO, error) {
	id := uuid.New()
	dto := repository.EmployeeDTO{
		ID:             id,
		OrganizationID: in.OrganizationID,
		EmployeeNo:     in.EmployeeNo,
		FullName:       in.FullName,
		DateOfBirth:    in.DateOfBirth,
		Email:          in.Email,
		HireDate:       platform.FormatDate(in.HireDate),
		Status:         "Active",
		RowVersion:     1,
	}
	if in.NationalID != "" {
		dto.NationalIDMasked = platform.MaskNationalID(in.NationalID)
		m.byHash[platform.HashLookup(m.key, in.NationalID)] = id
	}
	m.employees[id] = dto
	return dto, nil
}

func (m *memHRStore) GetByID(_ context.Context, _ []uuid.UUID, id uuid.UUID) (repository.EmployeeDTO, error) {
	d, ok := m.employees[id]
	if !ok {
		return repository.EmployeeDTO{}, httpx.NotFound("employee tidak ditemukan")
	}
	return d, nil
}

func (m *memHRStore) FindByNationalID(_ context.Context, _ []uuid.UUID, nik string) (repository.EmployeeDTO, error) {
	id, ok := m.byHash[platform.HashLookup(m.key, nik)]
	if !ok {
		return repository.EmployeeDTO{}, httpx.NotFound("employee tidak ditemukan")
	}
	return m.employees[id], nil
}

func (m *memHRStore) ListWithCount(_ context.Context, _ []uuid.UUID, status *string, _, _ int32) ([]repository.EmployeeDTO, int64, error) {
	var out []repository.EmployeeDTO
	for _, d := range m.employees {
		if status != nil && d.Status != *status {
			continue
		}
		out = append(out, d)
	}
	return out, int64(len(out)), nil
}

func (m *memHRStore) UpdateAndEmit(_ context.Context, _ []uuid.UUID, in repository.UpdateEmployeeInput) (repository.EmployeeDTO, error) {
	d, ok := m.employees[in.ID]
	if !ok {
		return repository.EmployeeDTO{}, httpx.NotFound("employee tidak ditemukan")
	}
	if d.RowVersion != in.RowVersion {
		return repository.EmployeeDTO{}, httpx.Conflict("VERSION_CONFLICT", "conflict")
	}
	d.FullName = in.FullName
	d.Email = in.Email
	if in.NationalID != "" {
		d.NationalIDMasked = platform.MaskNationalID(in.NationalID)
	}
	d.RowVersion++
	m.employees[in.ID] = d
	return d, nil
}

func (m *memHRStore) TerminateAndEmit(_ context.Context, _ []uuid.UUID, id uuid.UUID, version int32, _, _ uuid.UUID) (repository.EmployeeDTO, error) {
	d, ok := m.employees[id]
	if !ok {
		return repository.EmployeeDTO{}, httpx.NotFound("employee tidak ditemukan")
	}
	if d.Status != "Active" || d.RowVersion != version {
		return repository.EmployeeDTO{}, httpx.Conflict("INVALID_STATE", "conflict")
	}
	d.Status = "Terminated"
	d.RowVersion++
	m.employees[id] = d
	return d, nil
}

func (m *memHRStore) Delete(_ context.Context, _ []uuid.UUID, id uuid.UUID, version int32, _ uuid.UUID) error {
	d, ok := m.employees[id]
	if !ok {
		return httpx.NotFound("employee tidak ditemukan")
	}
	if d.RowVersion != version {
		return httpx.Conflict("VERSION_CONFLICT", "conflict")
	}
	delete(m.employees, id)
	return nil
}

func (m *memHRStore) CreatePosition(_ context.Context, _ []uuid.UUID, in hrdb.CreatePositionParams) (hrdb.CreatePositionRow, error) {
	return hrdb.CreatePositionRow{
		ID: uuid.New(), OrganizationID: in.OrganizationID, Code: in.Code, Title: in.Title, OrganizationRefID: in.OrganizationRefID, RowVersion: 1,
	}, nil
}

func (m *memHRStore) ListPositionsWithCount(_ context.Context, _ []uuid.UUID, _, _ int32) ([]hrdb.ListPositionsRow, int64, error) {
	return nil, 0, nil
}

func (m *memHRStore) CreateEmployment(_ context.Context, _ []uuid.UUID, in hrdb.CreateEmploymentParams) (hrdb.CreateEmploymentRow, error) {
	return hrdb.CreateEmploymentRow{
		ID: uuid.New(), OrganizationID: in.OrganizationID, EmployeeID: in.EmployeeID, PositionID: in.PositionID, EmploymentType: in.EmploymentType, EffectiveFrom: in.EffectiveFrom, RowVersion: 1,
	}, nil
}

func (m *memHRStore) ListEmploymentsByEmployee(_ context.Context, _ []uuid.UUID, _ uuid.UUID) ([]hrdb.ListEmploymentsByEmployeeRow, error) {
	return nil, nil
}

func (m *memHRStore) CreateCompensation(_ context.Context, _ []uuid.UUID, in hrdb.CreateCompensationParams) (hrdb.CreateCompensationRow, error) {
	return hrdb.CreateCompensationRow{
		ID: uuid.New(), OrganizationID: in.OrganizationID, EmployeeID: in.EmployeeID, ComponentCode: in.ComponentCode, Amount: in.Amount, EffectiveFrom: in.EffectiveFrom, RowVersion: 1,
	}, nil
}

func (m *memHRStore) ListCompensationsByEmployee(_ context.Context, _ []uuid.UUID, _ uuid.UUID) ([]hrdb.ListCompensationsByEmployeeRow, error) {
	return nil, nil
}

func doHRReq(t *testing.T, h http.Handler, method, path, body, perm, ifMatch string) *httptest.ResponseRecorder {
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

func TestHRCRUDMaskingBlindLookupAndRBAC(t *testing.T) {
	store := newMemHRStore()
	router := NewRouter(store, nil)

	// 1. RBAC reject write with hr:read
	rec := doHRReq(t, router, http.MethodPost, "/employees", `{"employeeNo":"E-1","fullName":"Budi","hireDate":"2026-01-05"}`, "hr:read", "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	// 2. Create employee with NIK + email + DOB
	rec = doHRReq(t, router, http.MethodPost, "/employees", `{"employeeNo":"EMP-01","fullName":"Budi Santoso","nationalId":"3174011234560009","dateOfBirth":"1992-08-17","email":"budi@sinergi.co.id","hireDate":"2026-01-05"}`, "hr:write", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d %s", rec.Code, rec.Body.String())
	}
	var created repository.EmployeeDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	if created.NationalIDMasked != "************0009" {
		t.Fatalf("expected masked NIK, got %+v", created)
	}

	// 3. Blind-index lookup by ?nationalId=...
	rec = doHRReq(t, router, http.MethodGet, "/employees?nationalId=3174011234560009", "", "hr:read", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 blind lookup, got %d %s", rec.Code, rec.Body.String())
	}

	// 4. Update & Terminate employee with If-Match
	rec = doHRReq(t, router, http.MethodPut, "/employees/"+created.ID.String(), `{"fullName":"Budi Santoso SE","email":"budi.se@sinergi.co.id"}`, "hr:write", `"1"`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 update, got %d %s", rec.Code, rec.Body.String())
	}
	rec = doHRReq(t, router, http.MethodPost, "/employees/"+created.ID.String()+"/terminate", "", "hr:write", `"2"`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 terminate, got %d %s", rec.Code, rec.Body.String())
	}
	var term repository.EmployeeDTO
	_ = json.Unmarshal(rec.Body.Bytes(), &term)
	if term.Status != "Terminated" || term.RowVersion != 3 {
		t.Fatalf("unexpected terminated state: %+v", term)
	}
}

