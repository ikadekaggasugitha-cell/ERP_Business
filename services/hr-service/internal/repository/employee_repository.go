package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/hr-service/internal/hrdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// EmployeeDTO represents an employee with application-layer PII decrypted
// for authorized readers and national ID masked per DBDD BAB 7.4.
type EmployeeDTO struct {
	ID               uuid.UUID  `json:"id"`
	OrganizationID   uuid.UUID  `json:"organizationId"`
	EmployeeNo       string     `json:"employeeNo"`
	FullName         string     `json:"fullName"`
	NationalIDMasked string     `json:"nationalIdMasked,omitempty"`
	DateOfBirth      string     `json:"dateOfBirth,omitempty"`
	Email            string     `json:"email,omitempty"`
	HireDate         string     `json:"hireDate"`
	Status           string     `json:"status"`
	UserRefID        *uuid.UUID `json:"userRefId,omitempty"`
	RowVersion       int32      `json:"rowVersion"`
}

type CreateEmployeeInput struct {
	OrganizationID uuid.UUID
	EmployeeNo     string
	FullName       string
	NationalID     string
	DateOfBirth    string
	Email          string
	HireDate       pgtype.Date
	UserRefID      *uuid.UUID
	CorrelationID  uuid.UUID
	CreatedBy      uuid.UUID
}

type UpdateEmployeeInput struct {
	ID            uuid.UUID
	RowVersion    int32
	FullName      string
	NationalID    string
	DateOfBirth   string
	Email         string
	UserRefID     *uuid.UUID
	CorrelationID uuid.UUID
	UpdatedBy     uuid.UUID
}

type encryptedEmployeeFields struct {
	FullName            string
	NationalID          *string
	Email               *string
	EncryptedFullName   *string
	EncryptedNationalID *string
	EncryptedDob        *string
	EncryptedEmail      *string
	NationalIDHash      *string
}

// EmployeeRepository persists HR employees, positions, employments, compensations,
// and outbox events while transparently encrypting PII fields at rest.
type EmployeeRepository struct {
	db     *platform.DB
	q      *hrdb.Queries
	piiKey []byte
}

func NewEmployeeRepository(db *platform.DB, piiKey []byte) *EmployeeRepository {
	var q *hrdb.Queries
	if db != nil && db.Pool != nil {
		q = hrdb.New(db.Pool)
	}
	return &EmployeeRepository{db: db, q: q, piiKey: piiKey}
}

func (r *EmployeeRepository) encryptPII(fullName, nationalID, dob, email string) (encryptedEmployeeFields, error) {
	var out encryptedEmployeeFields
	encName, err := platform.EncryptField(r.piiKey, strings.TrimSpace(fullName))
	if err != nil {
		return out, fmt.Errorf("encrypt full_name: %w", err)
	}
	out.FullName = encName
	out.EncryptedFullName = &encName

	if nik := strings.TrimSpace(nationalID); nik != "" {
		encNIK, err := platform.EncryptField(r.piiKey, nik)
		if err != nil {
			return out, fmt.Errorf("encrypt national_id: %w", err)
		}
		hash := platform.HashLookup(r.piiKey, nik)
		// Store HMAC blind-index in national_id too so uq_employee_national_id
		// enforces per-org uniqueness without storing plaintext NIK at rest.
		out.NationalID = &hash
		out.NationalIDHash = &hash
		out.EncryptedNationalID = &encNIK
	}
	if d := strings.TrimSpace(dob); d != "" {
		encDOB, err := platform.EncryptField(r.piiKey, d)
		if err != nil {
			return out, fmt.Errorf("encrypt date_of_birth: %w", err)
		}
		out.EncryptedDob = &encDOB
	}
	if em := strings.TrimSpace(email); em != "" {
		encEm, err := platform.EncryptField(r.piiKey, em)
		if err != nil {
			return out, fmt.Errorf("encrypt email: %w", err)
		}
		out.EncryptedEmail = &encEm
	}
	return out, nil
}

func (r *EmployeeRepository) decryptString(enc *string, legacyPlain string) (string, error) {
	if enc != nil && *enc != "" {
		return platform.DecryptField(r.piiKey, *enc)
	}
	return legacyPlain, nil
}

func (r *EmployeeRepository) toDTO(
	id, orgID uuid.UUID,
	empNo, rawFullName string,
	encName, encNIK, encDOB, encEmail *string,
	legacyDOB pgtype.Date,
	legacyEmail *string,
	hireDate pgtype.Date,
	status string,
	userRefID *uuid.UUID,
	rowVer int32,
) (EmployeeDTO, error) {
	fullName, err := r.decryptString(encName, rawFullName)
	if err != nil {
		return EmployeeDTO{}, fmt.Errorf("decrypt full_name: %w", err)
	}
	nikPlain, err := r.decryptString(encNIK, "")
	if err != nil {
		return EmployeeDTO{}, fmt.Errorf("decrypt national_id: %w", err)
	}
	dobPlain, err := r.decryptString(encDOB, platform.FormatDate(legacyDOB))
	if err != nil {
		return EmployeeDTO{}, fmt.Errorf("decrypt date_of_birth: %w", err)
	}
	fallbackEmail := ""
	if legacyEmail != nil {
		fallbackEmail = *legacyEmail
	}
	emailPlain, err := r.decryptString(encEmail, fallbackEmail)
	if err != nil {
		return EmployeeDTO{}, fmt.Errorf("decrypt email: %w", err)
	}
	dto := EmployeeDTO{
		ID:             id,
		OrganizationID: orgID,
		EmployeeNo:     empNo,
		FullName:       fullName,
		DateOfBirth:    dobPlain,
		Email:          emailPlain,
		HireDate:       platform.FormatDate(hireDate),
		Status:         status,
		UserRefID:      userRefID,
		RowVersion:     rowVer,
	}
	if nikPlain != "" {
		dto.NationalIDMasked = platform.MaskNationalID(nikPlain)
	}
	return dto, nil
}

func (r *EmployeeRepository) CreateAndEmit(ctx context.Context, scope []uuid.UUID, in CreateEmployeeInput) (EmployeeDTO, error) {
	enc, err := r.encryptPII(in.FullName, in.NationalID, in.DateOfBirth, in.Email)
	if err != nil {
		return EmployeeDTO{}, err
	}
	var out EmployeeDTO
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		row, err := qt.CreateEmployee(ctx, hrdb.CreateEmployeeParams{
			OrganizationID:      in.OrganizationID,
			EmployeeNo:          in.EmployeeNo,
			FullName:            enc.FullName,
			NationalID:          enc.NationalID,
			DateOfBirth:         pgtype.Date{},
			Email:               enc.Email,
			HireDate:            in.HireDate,
			UserRefID:           in.UserRefID,
			EncryptedFullName:   enc.EncryptedFullName,
			EncryptedNationalID: enc.EncryptedNationalID,
			EncryptedDob:        enc.EncryptedDob,
			EncryptedEmail:      enc.EncryptedEmail,
			NationalIDHash:      enc.NationalIDHash,
			CreatedBy:           in.CreatedBy,
		})
		if err != nil {
			return httpx.MapDBError(err, "employeeNo atau nationalId sudah digunakan")
		}
		payload, err := json.Marshal(map[string]any{
			"employeeId": row.ID.String(),
			"employeeNo": row.EmployeeNo,
			"status":     row.Status,
			"hireDate":   platform.FormatDate(row.HireDate),
		})
		if err != nil {
			return err
		}
		if err := qt.InsertOutbox(ctx, hrdb.InsertOutboxParams{
			AggregateType:  "Employee",
			AggregateID:    row.ID,
			EventType:      "hr.employee.updated",
			Payload:        payload,
			OrganizationID: row.OrganizationID,
			CorrelationID:  in.CorrelationID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}
		out, err = r.toDTO(
			row.ID, row.OrganizationID, row.EmployeeNo, row.FullName,
			row.EncryptedFullName, row.EncryptedNationalID, row.EncryptedDob, row.EncryptedEmail,
			row.DateOfBirth, row.Email, row.HireDate, row.Status, row.UserRefID, row.RowVersion,
		)
		return err
	})
	return out, err
}

func (r *EmployeeRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (EmployeeDTO, error) {
	var out EmployeeDTO
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		row, err := r.q.WithTx(tx).GetEmployee(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("employee tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		out, err = r.toDTO(
			row.ID, row.OrganizationID, row.EmployeeNo, row.FullName,
			row.EncryptedFullName, row.EncryptedNationalID, row.EncryptedDob, row.EncryptedEmail,
			row.DateOfBirth, row.Email, row.HireDate, row.Status, row.UserRefID, row.RowVersion,
		)
		return err
	})
	return out, err
}

func (r *EmployeeRepository) FindByNationalID(ctx context.Context, scope []uuid.UUID, nationalID string) (EmployeeDTO, error) {
	var out EmployeeDTO
	hash := platform.HashLookup(r.piiKey, nationalID)
	if hash == "" {
		return out, httpx.BadRequest("nationalId tidak boleh kosong")
	}
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		row, err := r.q.WithTx(tx).GetEmployeeByNationalIDHash(ctx, &hash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("employee tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		out, err = r.toDTO(
			row.ID, row.OrganizationID, row.EmployeeNo, row.FullName,
			row.EncryptedFullName, row.EncryptedNationalID, row.EncryptedDob, row.EncryptedEmail,
			row.DateOfBirth, row.Email, row.HireDate, row.Status, row.UserRefID, row.RowVersion,
		)
		return err
	})
	return out, err
}

func (r *EmployeeRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]EmployeeDTO, int64, error) {
	var items []EmployeeDTO
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		var err error
		total, err = qt.CountEmployees(ctx, status)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		rows, err := qt.ListEmployees(ctx, hrdb.ListEmployeesParams{
			Limit:  limit,
			Offset: offset,
			Status: status,
		})
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		items = make([]EmployeeDTO, 0, len(rows))
		for _, row := range rows {
			dto, err := r.toDTO(
				row.ID, row.OrganizationID, row.EmployeeNo, row.FullName,
				row.EncryptedFullName, row.EncryptedNationalID, row.EncryptedDob, row.EncryptedEmail,
				row.DateOfBirth, row.Email, row.HireDate, row.Status, row.UserRefID, row.RowVersion,
			)
			if err != nil {
				return err
			}
			items = append(items, dto)
		}
		return nil
	})
	return items, total, err
}

func (r *EmployeeRepository) UpdateAndEmit(ctx context.Context, scope []uuid.UUID, in UpdateEmployeeInput) (EmployeeDTO, error) {
	enc, err := r.encryptPII(in.FullName, in.NationalID, in.DateOfBirth, in.Email)
	if err != nil {
		return EmployeeDTO{}, err
	}
	var out EmployeeDTO
	err = r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		cur, err := qt.GetEmployee(ctx, in.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("employee tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if cur.RowVersion != in.RowVersion {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		// Preserve existing PII/metadata when omitted on update (GET returns masked nationalId).
		if strings.TrimSpace(in.NationalID) == "" {
			enc.NationalID = cur.NationalID
			enc.EncryptedNationalID = cur.EncryptedNationalID
			enc.NationalIDHash = cur.NationalIDHash
		}
		if strings.TrimSpace(in.DateOfBirth) == "" {
			enc.EncryptedDob = cur.EncryptedDob
		}
		if strings.TrimSpace(in.Email) == "" {
			enc.Email = cur.Email
			enc.EncryptedEmail = cur.EncryptedEmail
		}
		userRef := in.UserRefID
		if userRef == nil {
			userRef = cur.UserRefID
		}
		if _, err := qt.UpdateEmployee(ctx, hrdb.UpdateEmployeeParams{
			ID:                  in.ID,
			RowVersion:          in.RowVersion,
			FullName:            enc.FullName,
			NationalID:          enc.NationalID,
			Email:               enc.Email,
			EncryptedFullName:   enc.EncryptedFullName,
			EncryptedNationalID: enc.EncryptedNationalID,
			EncryptedDob:        enc.EncryptedDob,
			EncryptedEmail:      enc.EncryptedEmail,
			NationalIDHash:      enc.NationalIDHash,
			UserRefID:           userRef,
			UpdatedBy:           &in.UpdatedBy,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("VERSION_CONFLICT", "employee berubah saat diperbarui")
			}
			return httpx.MapDBError(err, "nationalId sudah digunakan")
		}
		row, err := qt.GetEmployee(ctx, in.ID)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		payload, err := json.Marshal(map[string]any{
			"employeeId": row.ID.String(),
			"employeeNo": row.EmployeeNo,
			"status":     row.Status,
			"hireDate":   platform.FormatDate(row.HireDate),
		})
		if err != nil {
			return err
		}
		if err := qt.InsertOutbox(ctx, hrdb.InsertOutboxParams{
			AggregateType:  "Employee",
			AggregateID:    row.ID,
			EventType:      "hr.employee.updated",
			Payload:        payload,
			OrganizationID: row.OrganizationID,
			CorrelationID:  in.CorrelationID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}
		out, err = r.toDTO(
			row.ID, row.OrganizationID, row.EmployeeNo, row.FullName,
			row.EncryptedFullName, row.EncryptedNationalID, row.EncryptedDob, row.EncryptedEmail,
			row.DateOfBirth, row.Email, row.HireDate, row.Status, row.UserRefID, row.RowVersion,
		)
		return err
	})
	return out, err
}

func (r *EmployeeRepository) TerminateAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy, corrID uuid.UUID) (EmployeeDTO, error) {
	var out EmployeeDTO
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		cur, err := qt.GetEmployee(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("employee tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if cur.Status != "Active" {
			return httpx.Conflict("INVALID_STATE", "hanya employee berstatus Active yang dapat diterminasi")
		}
		if cur.RowVersion != version {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		newVer, err := qt.TerminateEmployee(ctx, hrdb.TerminateEmployeeParams{
			ID:         id,
			RowVersion: version,
			UpdatedBy:  &updatedBy,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("VERSION_CONFLICT", "employee gagal diterminasi karena perubahan paralel")
			}
			return httpx.MapDBError(err, "")
		}
		cur.Status = "Terminated"
		cur.RowVersion = newVer

		payload, err := json.Marshal(map[string]any{
			"employeeId": cur.ID.String(),
			"employeeNo": cur.EmployeeNo,
			"status":     "Terminated",
			"hireDate":   platform.FormatDate(cur.HireDate),
		})
		if err != nil {
			return err
		}
		if err := qt.InsertOutbox(ctx, hrdb.InsertOutboxParams{
			AggregateType:  "Employee",
			AggregateID:    cur.ID,
			EventType:      "hr.employee.updated",
			Payload:        payload,
			OrganizationID: cur.OrganizationID,
			CorrelationID:  corrID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}
		out, err = r.toDTO(
			cur.ID, cur.OrganizationID, cur.EmployeeNo, cur.FullName,
			cur.EncryptedFullName, cur.EncryptedNationalID, cur.EncryptedDob, cur.EncryptedEmail,
			cur.DateOfBirth, cur.Email, cur.HireDate, cur.Status, cur.UserRefID, cur.RowVersion,
		)
		return err
	})
	return out, err
}

func (r *EmployeeRepository) Delete(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy uuid.UUID) error {
	return r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		cur, err := qt.GetEmployee(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("employee tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if cur.Status == "Active" {
			return httpx.Conflict("INVALID_STATE", "karyawan berstatus Active harus diterminasi terlebih dahulu sebelum dihapus")
		}
		if cur.RowVersion != version {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		activeEmp, err := qt.CountActiveEmploymentsByEmployee(ctx, id)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		if activeEmp > 0 {
			return httpx.Conflict("HAS_ACTIVE_EMPLOYMENT", "karyawan masih memiliki penugasan kerja aktif")
		}
		n, err := qt.SoftDeleteEmployee(ctx, hrdb.SoftDeleteEmployeeParams{
			ID:         id,
			RowVersion: version,
			UpdatedBy:  &updatedBy,
		})
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		if n == 0 {
			return httpx.Conflict("VERSION_CONFLICT", "employee gagal dihapus")
		}
		return nil
	})
}

func (r *EmployeeRepository) CreatePosition(ctx context.Context, scope []uuid.UUID, in hrdb.CreatePositionParams) (hrdb.CreatePositionRow, error) {
	var out hrdb.CreatePositionRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreatePosition(ctx, in)
		return httpx.MapDBError(e, "kode posisi sudah digunakan")
	})
	return out, err
}

func (r *EmployeeRepository) ListPositionsWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]hrdb.ListPositionsRow, int64, error) {
	var items []hrdb.ListPositionsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		var err error
		total, err = qt.CountPositions(ctx)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		items, err = qt.ListPositions(ctx, hrdb.ListPositionsParams{Limit: limit, Offset: offset})
		return httpx.MapDBError(err, "")
	})
	return items, total, err
}

func (r *EmployeeRepository) CreateEmployment(ctx context.Context, scope []uuid.UUID, in hrdb.CreateEmploymentParams) (hrdb.CreateEmploymentRow, error) {
	var out hrdb.CreateEmploymentRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).CreateEmployment(ctx, in)
		return httpx.MapDBError(e, "")
	})
	return out, err
}

func (r *EmployeeRepository) ListEmploymentsByEmployee(ctx context.Context, scope []uuid.UUID, employeeID uuid.UUID) ([]hrdb.ListEmploymentsByEmployeeRow, error) {
	var items []hrdb.ListEmploymentsByEmployeeRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		items, e = r.q.WithTx(tx).ListEmploymentsByEmployee(ctx, employeeID)
		return httpx.MapDBError(e, "")
	})
	return items, err
}

func (r *EmployeeRepository) CreateCompensation(ctx context.Context, scope []uuid.UUID, in hrdb.CreateCompensationParams) (hrdb.CreateCompensationRow, error) {
	var out hrdb.CreateCompensationRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		emp, e := qt.GetEmployee(ctx, in.EmployeeID)
		if e != nil {
			if errors.Is(e, pgx.ErrNoRows) {
				return httpx.NotFound("employee tidak ditemukan")
			}
			return httpx.MapDBError(e, "")
		}
		out, e = qt.CreateCompensation(ctx, in)
		if e != nil {
			return httpx.MapDBError(e, "")
		}
		payload, e := json.Marshal(map[string]any{
			"employeeId":    emp.ID.String(),
			"employeeNo":    emp.EmployeeNo,
			"status":        emp.Status,
			"componentCode": out.ComponentCode,
			"amount":        platform.FormatNumeric(out.Amount, 4),
			"effectiveFrom": platform.FormatDate(out.EffectiveFrom),
		})
		if e != nil {
			return e
		}
		if e := qt.InsertOutbox(ctx, hrdb.InsertOutboxParams{
			AggregateType:  "Employee",
			AggregateID:    emp.ID,
			EventType:      "hr.employee.updated",
			Payload:        payload,
			OrganizationID: emp.OrganizationID,
			CorrelationID:  uuid.New(),
		}); e != nil {
			return httpx.MapDBError(e, "")
		}
		return nil
	})
	return out, err
}

func (r *EmployeeRepository) ListCompensationsByEmployee(ctx context.Context, scope []uuid.UUID, employeeID uuid.UUID) ([]hrdb.ListCompensationsByEmployeeRow, error) {
	var items []hrdb.ListCompensationsByEmployeeRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		items, e = r.q.WithTx(tx).ListCompensationsByEmployee(ctx, employeeID)
		return httpx.MapDBError(e, "")
	})
	return items, err
}

func (r *EmployeeRepository) DrainAndPublish(ctx context.Context, batch int32, pub *eventbus.Publisher) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	qt := r.q.WithTx(tx)
	rows, err := qt.DrainOutbox(ctx, batch)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	published := 0
	var pubErr error
	for _, row := range rows {
		if e := pub.Publish(ctx, eventbus.Envelope{
			EventID:        row.ID.String(),
			EventType:      row.EventType,
			AggregateType:  row.AggregateType,
			AggregateID:    row.AggregateID.String(),
			OccurredAt:     row.CreatedAt.Time,
			CorrelationID:  row.CorrelationID.String(),
			OrganizationID: row.OrganizationID.String(),
			SchemaVersion:  "v1",
			Data:           json.RawMessage(row.Payload),
		}); e != nil {
			pubErr = e
			break
		}
		if e := qt.MarkOutboxPublished(ctx, hrdb.MarkOutboxPublishedParams{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
		}); e != nil {
			pubErr = e
			break
		}
		published++
	}
	if published > 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit outbox tx: %w", err)
		}
		committed = true
	}
	return published, pubErr
}

var _ = time.Second
