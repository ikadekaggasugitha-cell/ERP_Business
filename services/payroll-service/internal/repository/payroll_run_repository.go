package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/calc"
	"github.com/sinergiinvestasiproperti/erp/services/payroll-service/internal/payrolldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type PayrollResultLineDTO struct {
	ID            uuid.UUID `json:"id"`
	ComponentCode string    `json:"componentCode"`
	Amount        string    `json:"amount"`
}

type StatutoryContributionDTO struct {
	ID               uuid.UUID `json:"id"`
	ContributionType string    `json:"contributionType"`
	EmployeeAmount   string    `json:"employeeAmount"`
	EmployerAmount   string    `json:"employerAmount"`
}

type PayrollResultDTO struct {
	ID                     uuid.UUID                  `json:"id"`
	EmployeeID             uuid.UUID                  `json:"employeeId"`
	GrossAmount            string                     `json:"grossAmount"`
	DeductionAmount        string                     `json:"deductionAmount"`
	TaxAmount              string                     `json:"taxAmount"`
	NetAmount              string                     `json:"netAmount"`
	RowVersion             int32                      `json:"rowVersion"`
	Lines                  []PayrollResultLineDTO     `json:"lines,omitempty"`
	StatutoryContributions []StatutoryContributionDTO `json:"statutoryContributions,omitempty"`
}

type PayrollRunDetail struct {
	ID             uuid.UUID          `json:"id"`
	OrganizationID uuid.UUID          `json:"organizationId"`
	RunNo          string             `json:"runNo"`
	PeriodMonth    int32              `json:"periodMonth"`
	PeriodYear     int32              `json:"periodYear"`
	Status         string             `json:"status"`
	GLJournalRef   string             `json:"glJournalRef,omitempty"`
	CorrelationID  uuid.UUID          `json:"correlationId"`
	RowVersion     int32              `json:"rowVersion"`
	Results        []PayrollResultDTO `json:"results,omitempty"`
}

type PayrollRunRepository struct {
	db     *platform.DB
	q      *payrolldb.Queries
	piiKey []byte
}

func NewPayrollRunRepository(db *platform.DB, piiKey []byte) *PayrollRunRepository {
	var q *payrolldb.Queries
	if db != nil && db.Pool != nil {
		q = payrolldb.New(db.Pool)
	}
	return &PayrollRunRepository{db: db, q: q, piiKey: piiKey}
}

func (r *PayrollRunRepository) CreateRun(ctx context.Context, scope []uuid.UUID, orgID uuid.UUID, runNo string, month, year int32, corrID, createdBy uuid.UUID) (PayrollRunDetail, error) {
	var out PayrollRunDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		row, err := r.q.WithTx(tx).CreatePayrollRun(ctx, payrolldb.CreatePayrollRunParams{
			OrganizationID: orgID,
			RunNo:          runNo,
			PeriodMonth:    month,
			PeriodYear:     year,
			CorrelationID:  corrID,
			CreatedBy:      createdBy,
		})
		if err != nil {
			return httpx.MapDBError(err, "runNo sudah digunakan")
		}
		out = PayrollRunDetail{
			ID:             row.ID,
			OrganizationID: row.OrganizationID,
			RunNo:          row.RunNo,
			PeriodMonth:    row.PeriodMonth,
			PeriodYear:     row.PeriodYear,
			Status:         row.Status,
			CorrelationID:  row.CorrelationID,
			RowVersion:     row.RowVersion,
		}
		if row.GlJournalRef != nil {
			out.GLJournalRef = *row.GlJournalRef
		}
		return nil
	})
	return out, err
}

func (r *PayrollRunRepository) loadDetail(ctx context.Context, qt *payrolldb.Queries, run payrolldb.GetPayrollRunRow) (PayrollRunDetail, error) {
	out := PayrollRunDetail{
		ID:             run.ID,
		OrganizationID: run.OrganizationID,
		RunNo:          run.RunNo,
		PeriodMonth:    run.PeriodMonth,
		PeriodYear:     run.PeriodYear,
		Status:         run.Status,
		CorrelationID:  run.CorrelationID,
		RowVersion:     run.RowVersion,
	}
	if run.GlJournalRef != nil {
		out.GLJournalRef = *run.GlJournalRef
	}
	resRows, err := qt.ListPayrollResultsByRun(ctx, run.ID)
	if err != nil {
		return out, httpx.MapDBError(err, "")
	}
	lineRows, err := qt.ListPayrollResultLinesByRun(ctx, run.ID)
	if err != nil {
		return out, httpx.MapDBError(err, "")
	}
	contribRows, err := qt.ListStatutoryContributionsByRun(ctx, run.ID)
	if err != nil {
		return out, httpx.MapDBError(err, "")
	}

	lineMap := make(map[uuid.UUID][]PayrollResultLineDTO)
	for _, l := range lineRows {
		lineMap[l.PayrollResultID] = append(lineMap[l.PayrollResultID], PayrollResultLineDTO{
			ID:            l.ID,
			ComponentCode: l.ComponentCode,
			Amount:        platform.FormatNumeric(l.Amount, 4),
		})
	}
	contribMap := make(map[uuid.UUID][]StatutoryContributionDTO)
	for _, c := range contribRows {
		contribMap[c.PayrollResultID] = append(contribMap[c.PayrollResultID], StatutoryContributionDTO{
			ID:               c.ID,
			ContributionType: c.ContributionType,
			EmployeeAmount:   platform.FormatNumeric(c.EmployeeAmount, 4),
			EmployerAmount:   platform.FormatNumeric(c.EmployerAmount, 4),
		})
	}
	results := make([]PayrollResultDTO, 0, len(resRows))
	for _, row := range resRows {
		grossStr := platform.FormatNumeric(row.GrossAmount, 4)
		taxStr := platform.FormatNumeric(row.TaxAmount, 4)
		netStr := platform.FormatNumeric(row.NetAmount, 4)
		if row.EncryptedGross != nil && *row.EncryptedGross != "" {
			if pt, err := platform.DecryptField(r.piiKey, *row.EncryptedGross); err == nil {
				grossStr = pt
			} else {
				return out, fmt.Errorf("decrypt gross_amount: %w", err)
			}
		}
		if row.EncryptedTax != nil && *row.EncryptedTax != "" {
			if pt, err := platform.DecryptField(r.piiKey, *row.EncryptedTax); err == nil {
				taxStr = pt
			} else {
				return out, fmt.Errorf("decrypt tax_amount: %w", err)
			}
		}
		if row.EncryptedNet != nil && *row.EncryptedNet != "" {
			if pt, err := platform.DecryptField(r.piiKey, *row.EncryptedNet); err == nil {
				netStr = pt
			} else {
				return out, fmt.Errorf("decrypt net_amount: %w", err)
			}
		}
		results = append(results, PayrollResultDTO{
			ID:                     row.ID,
			EmployeeID:             row.EmployeeID,
			GrossAmount:            grossStr,
			DeductionAmount:        platform.FormatNumeric(row.DeductionAmount, 4),
			TaxAmount:              taxStr,
			NetAmount:              netStr,
			RowVersion:             row.RowVersion,
			Lines:                  lineMap[row.ID],
			StatutoryContributions: contribMap[row.ID],
		})
	}
	out.Results = results
	return out, nil
}

func (r *PayrollRunRepository) GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (PayrollRunDetail, error) {
	var out PayrollRunDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		run, err := qt.GetPayrollRun(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("payroll run tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		out, err = r.loadDetail(ctx, qt, run)
		return err
	})
	return out, err
}

func (r *PayrollRunRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]payrolldb.ListPayrollRunsRow, int64, error) {
	var items []payrolldb.ListPayrollRunsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		var err error
		total, err = qt.CountPayrollRuns(ctx, status)
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		items, err = qt.ListPayrollRuns(ctx, payrolldb.ListPayrollRunsParams{
			Limit:  limit,
			Offset: offset,
			Status: status,
		})
		return httpx.MapDBError(err, "")
	})
	return items, total, err
}

func (r *PayrollRunRepository) EncryptAmounts(gross, tax, net string) (encGross, encTax, encNet string, err error) {
	if encGross, err = platform.EncryptField(r.piiKey, gross); err != nil {
		return "", "", "", err
	}
	if encTax, err = platform.EncryptField(r.piiKey, tax); err != nil {
		return "", "", "", err
	}
	if encNet, err = platform.EncryptField(r.piiKey, net); err != nil {
		return "", "", "", err
	}
	return encGross, encTax, encNet, nil
}

func (r *PayrollRunRepository) CalculateAndEmit(
	ctx context.Context,
	scope []uuid.UUID,
	id uuid.UUID,
	version int32,
	summaryGross *big.Rat,
	employees []calc.EmployeeInput,
	updatedBy uuid.UUID,
) (PayrollRunDetail, error) {
	var out PayrollRunDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		run, err := qt.GetPayrollRun(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("payroll run tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if run.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "hanya payroll run berstatus Draft yang dapat dihitung")
		}
		if run.RowVersion != version {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		if len(employees) == 0 {
			return httpx.Unprocessable("EMPTY_EMPLOYEES", "minimal 1 karyawan wajib disertakan dalam perhitungan payroll")
		}
		if err := qt.DeletePayrollResultsByRun(ctx, id); err != nil {
			return httpx.MapDBError(err, "")
		}

		zeroNum, _ := platform.ParseNumeric("0.0000")
		piiVer := "v1"
		grossSum := new(big.Rat)
		taxSum := new(big.Rat)
		netSum := new(big.Rat)

		for _, emp := range employees {
			comp, err := calc.ComputeEmployee(emp)
			if err != nil {
				return httpx.Unprocessable("CALCULATION_ERROR", err.Error())
			}
			gStr := platform.FormatNumeric(comp.Gross, 4)
			tStr := platform.FormatNumeric(comp.TaxPPh21, 4)
			nStr := platform.FormatNumeric(comp.Net, 4)
			grossSum.Add(grossSum, platform.NumericToRat(comp.Gross))
			taxSum.Add(taxSum, platform.NumericToRat(comp.TaxPPh21))
			netSum.Add(netSum, platform.NumericToRat(comp.Net))

			encG, encT, encN, err := r.EncryptAmounts(gStr, tStr, nStr)
			if err != nil {
				return err
			}
			resRow, err := qt.CreatePayrollResult(ctx, payrolldb.CreatePayrollResultParams{
				OrganizationID:  run.OrganizationID,
				PayrollRunID:    run.ID,
				EmployeeID:      emp.EmployeeID,
				GrossAmount:     zeroNum,
				DeductionAmount: comp.Deduction,
				TaxAmount:       zeroNum,
				NetAmount:       zeroNum,
				EncryptedGross:  &encG,
				EncryptedNet:    &encN,
				EncryptedTax:    &encT,
				PiiKeyVersion:   &piiVer,
				CreatedBy:       updatedBy,
			})
			if err != nil {
				return httpx.MapDBError(err, "duplikat employeeId pada payroll run")
			}

			linePairs := []struct {
				code string
				amt  pgtype.Numeric
			}{
				{code: "BASIC", amt: comp.Basic},
				{code: "ALLOWANCE", amt: comp.Allowance},
				{code: "BONUS", amt: comp.Bonus},
				{code: "BPJS_KES_EMP", amt: comp.BPJSKesEmp},
				{code: "BPJS_JHT_EMP", amt: comp.BPJSJHTEmp},
				{code: "BPJS_JP_EMP", amt: comp.BPJSJPEmp},
				{code: "PPH21", amt: comp.TaxPPh21},
			}
			for _, lp := range linePairs {
				if rVal := platform.NumericToRat(lp.amt); rVal == nil || rVal.Sign() == 0 {
					continue
				}
				if err := qt.CreatePayrollResultLine(ctx, payrolldb.CreatePayrollResultLineParams{
					OrganizationID:  run.OrganizationID,
					PayrollResultID: resRow.ID,
					ComponentCode:   lp.code,
					Amount:          lp.amt,
					CreatedBy:       updatedBy,
				}); err != nil {
					return httpx.MapDBError(err, "")
				}
			}

			statPairs := []struct {
				typ string
				emp pgtype.Numeric
				emr pgtype.Numeric
			}{
				{typ: "BPJS-Kesehatan", emp: comp.BPJSKesEmp, emr: comp.BPJSKesEmployer},
				{typ: "BPJS-TK-JHT", emp: comp.BPJSJHTEmp, emr: comp.BPJSJHTEmployer},
				{typ: "BPJS-TK-JP", emp: comp.BPJSJPEmp, emr: comp.BPJSJPEmployer},
				{typ: "BPJS-TK-JKK-JKM", emp: zeroNum, emr: comp.BPJSJKKJKMEmp},
			}
			for _, sp := range statPairs {
				if err := qt.CreateStatutoryContribution(ctx, payrolldb.CreateStatutoryContributionParams{
					OrganizationID:   run.OrganizationID,
					PayrollResultID:  resRow.ID,
					ContributionType: sp.typ,
					EmployeeAmount:   sp.emp,
					EmployerAmount:   sp.emr,
					CreatedBy:        updatedBy,
				}); err != nil {
					return httpx.MapDBError(err, "")
				}
			}
		}

		newVer, err := qt.CalculatePayrollRun(ctx, payrolldb.CalculatePayrollRunParams{
			ID:         run.ID,
			RowVersion: version,
			UpdatedBy:  &updatedBy,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("VERSION_CONFLICT", "payroll run gagal diperbarui karena perubahan paralel")
			}
			return httpx.MapDBError(err, "")
		}
		run.Status = "Calculated"
		run.RowVersion = newVer

		grossNum, _ := platform.RatToNumeric(grossSum, 4)
		taxNum, _ := platform.RatToNumeric(taxSum, 4)
		netNum, _ := platform.RatToNumeric(netSum, 4)
		totGrossStr := platform.FormatNumeric(grossNum, 4)
		totTaxStr := platform.FormatNumeric(taxNum, 4)
		totNetStr := platform.FormatNumeric(netNum, 4)

		payload, err := json.Marshal(map[string]any{
			"runNo":         run.RunNo,
			"period":        fmt.Sprintf("%d-%d", run.PeriodYear, run.PeriodMonth),
			"periodCode":    fmt.Sprintf("%04d-%02d", run.PeriodYear, run.PeriodMonth),
			"status":        "Calculated",
			"totalAmount":   totGrossStr,
			"totalGross":    totGrossStr,
			"totalTax":      totTaxStr,
			"totalNet":      totNetStr,
			"employeeCount": len(employees),
		})
		if err != nil {
			return err
		}
		if err := qt.InsertOutbox(ctx, payrolldb.InsertOutboxParams{
			AggregateType:  "PayrollRun",
			AggregateID:    run.ID,
			EventType:      "payroll.run.calculated",
			Payload:        payload,
			OrganizationID: run.OrganizationID,
			CorrelationID:  run.CorrelationID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}
		out, err = r.loadDetail(ctx, qt, run)
		return err
	})
	return out, err
}

func (r *PayrollRunRepository) ApproveAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy, corrID uuid.UUID) (PayrollRunDetail, error) {
	var out PayrollRunDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		run, err := qt.GetPayrollRun(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("payroll run tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if run.Status != "Calculated" {
			return httpx.Conflict("INVALID_STATE", "hanya payroll run berstatus Calculated yang dapat disetujui")
		}
		if run.RowVersion != version {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		newVer, err := qt.ApprovePayrollRun(ctx, payrolldb.ApprovePayrollRunParams{
			ID:         id,
			RowVersion: version,
			UpdatedBy:  &updatedBy,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("VERSION_CONFLICT", "payroll run gagal disetujui")
			}
			return httpx.MapDBError(err, "")
		}
		run.Status = "Approved"
		run.RowVersion = newVer

		payload, _ := json.Marshal(map[string]any{"runNo": run.RunNo, "status": "Approved"})
		if err := qt.InsertOutbox(ctx, payrolldb.InsertOutboxParams{
			AggregateType:  "PayrollRun",
			AggregateID:    run.ID,
			EventType:      "payroll.run.approved",
			Payload:        payload,
			OrganizationID: run.OrganizationID,
			CorrelationID:  corrID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}
		out, err = r.loadDetail(ctx, qt, run)
		return err
	})
	return out, err
}

func sumPayrollResultTotals(results []PayrollResultDTO) (grossStr, taxStr, netStr string, err error) {
	grossSum := new(big.Rat)
	taxSum := new(big.Rat)
	netSum := new(big.Rat)
	for _, res := range results {
		gNum, e := platform.ParseOptionalNumeric(res.GrossAmount)
		if e != nil {
			return "", "", "", fmt.Errorf("invalid grossAmount %q: %w", res.GrossAmount, e)
		}
		tNum, e := platform.ParseOptionalNumeric(res.TaxAmount)
		if e != nil {
			return "", "", "", fmt.Errorf("invalid taxAmount %q: %w", res.TaxAmount, e)
		}
		nNum, e := platform.ParseOptionalNumeric(res.NetAmount)
		if e != nil {
			return "", "", "", fmt.Errorf("invalid netAmount %q: %w", res.NetAmount, e)
		}
		grossSum.Add(grossSum, platform.NumericToRat(gNum))
		taxSum.Add(taxSum, platform.NumericToRat(tNum))
		netSum.Add(netSum, platform.NumericToRat(nNum))
	}
	gOut, _ := platform.RatToNumeric(grossSum, 4)
	tOut, _ := platform.RatToNumeric(taxSum, 4)
	nOut, _ := platform.RatToNumeric(netSum, 4)
	return platform.FormatNumeric(gOut, 4), platform.FormatNumeric(tOut, 4), platform.FormatNumeric(nOut, 4), nil
}

func (r *PayrollRunRepository) PostAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, glJournalRef string, updatedBy, corrID uuid.UUID) (PayrollRunDetail, error) {
	var out PayrollRunDetail
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		run, err := qt.GetPayrollRun(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("payroll run tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if run.Status != "Approved" {
			return httpx.Conflict("INVALID_STATE", "hanya payroll run berstatus Approved yang dapat diposting")
		}
		if run.RowVersion != version {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		if glJournalRef == "" {
			glJournalRef = "AUTO-" + run.RunNo
		}
		newVer, err := qt.PostPayrollRun(ctx, payrolldb.PostPayrollRunParams{
			ID:           id,
			RowVersion:   version,
			UpdatedBy:    &updatedBy,
			GlJournalRef: &glJournalRef,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.Conflict("VERSION_CONFLICT", "payroll run gagal diposting")
			}
			return httpx.MapDBError(err, "")
		}
		run.Status = "Posted"
		run.GlJournalRef = &glJournalRef
		run.RowVersion = newVer

		out, err = r.loadDetail(ctx, qt, run)
		if err != nil {
			return err
		}
		totGrossStr, totTaxStr, totNetStr, err := sumPayrollResultTotals(out.Results)
		if err != nil {
			return err
		}

		postedPayload, err := json.Marshal(map[string]any{
			"runNo":         run.RunNo,
			"period":        fmt.Sprintf("%d-%d", run.PeriodYear, run.PeriodMonth),
			"periodCode":    fmt.Sprintf("%04d-%02d", run.PeriodYear, run.PeriodMonth),
			"status":        "Posted",
			"glJournalRef":  glJournalRef,
			"totalAmount":   totGrossStr,
			"totalGross":    totGrossStr,
			"totalTax":      totTaxStr,
			"totalNet":      totNetStr,
			"employeeCount": len(out.Results),
		})
		if err != nil {
			return err
		}
		if err := qt.InsertOutbox(ctx, payrolldb.InsertOutboxParams{
			AggregateType:  "PayrollRun",
			AggregateID:    run.ID,
			EventType:      "payroll.run.posted",
			Payload:        postedPayload,
			OrganizationID: run.OrganizationID,
			CorrelationID:  corrID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}
		if err := qt.InsertOutbox(ctx, payrolldb.InsertOutboxParams{
			AggregateType:  "PayrollRun",
			AggregateID:    run.ID,
			EventType:      "payroll.run.completed",
			Payload:        postedPayload,
			OrganizationID: run.OrganizationID,
			CorrelationID:  corrID,
		}); err != nil {
			return httpx.MapDBError(err, "")
		}
		return nil
	})
	return out, err
}

func (r *PayrollRunRepository) DeleteDraft(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy uuid.UUID) error {
	return r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		qt := r.q.WithTx(tx)
		run, err := qt.GetPayrollRun(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.NotFound("payroll run tidak ditemukan")
			}
			return httpx.MapDBError(err, "")
		}
		if run.Status != "Draft" {
			return httpx.Conflict("INVALID_STATE", "hanya payroll run Draft yang dapat dihapus")
		}
		if run.RowVersion != version {
			return httpx.Conflict("VERSION_CONFLICT", "rowVersion tidak cocok")
		}
		n, err := qt.SoftDeleteDraftPayrollRun(ctx, payrolldb.SoftDeleteDraftPayrollRunParams{
			ID:         id,
			RowVersion: version,
			UpdatedBy:  &updatedBy,
		})
		if err != nil {
			return httpx.MapDBError(err, "")
		}
		if n == 0 {
			return httpx.Conflict("VERSION_CONFLICT", "payroll run gagal dihapus")
		}
		return nil
	})
}

func (r *PayrollRunRepository) DrainAndPublish(ctx context.Context, batch int32, pub *eventbus.Publisher) (int, error) {
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
		if e := qt.MarkOutboxPublished(ctx, payrolldb.MarkOutboxPublishedParams{
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
