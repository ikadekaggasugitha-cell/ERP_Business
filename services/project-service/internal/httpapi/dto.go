package httpapi

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/project-service/internal/prjdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func fmtOptTS(ts pgtype.Timestamptz) *string {
	if !ts.Valid {
		return nil
	}
	s := ts.Time.UTC().Format(time.RFC3339)
	return &s
}

func fmtOptDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := platform.FormatDate(d)
	return &s
}

type projectDTO struct {
	ID              uuid.UUID `json:"id"`
	OrganizationID  uuid.UUID `json:"organizationId"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	ProjectType     string    `json:"projectType"`
	OrganizationRef uuid.UUID `json:"organizationRef"`
	CurrencyCode    string    `json:"currencyCode"`
	StartDate       *string   `json:"startDate,omitempty"`
	EndDate         *string   `json:"endDate,omitempty"`
	Status          string    `json:"status"`
	IsSystem        bool      `json:"isSystem"`
	CreatedAt       *string   `json:"createdAt,omitempty"`
	UpdatedAt       *string   `json:"updatedAt,omitempty"`
	RowVersion      int32     `json:"rowVersion"`
}

func toProjectDTO(r prjdb.PrjProject) projectDTO {
	return projectDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, Code: r.Code, Name: r.Name,
		ProjectType: r.ProjectType, OrganizationRef: r.OrganizationRef,
		CurrencyCode: r.CurrencyCode, StartDate: fmtOptDate(r.StartDate), EndDate: fmtOptDate(r.EndDate),
		Status: r.Status, IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type wbsDTO struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	ProjectID      uuid.UUID  `json:"projectId"`
	ParentID       *uuid.UUID `json:"parentId,omitempty"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	LevelNo        int32      `json:"levelNo"`
	IsSystem       bool       `json:"isSystem"`
	CreatedAt      *string    `json:"createdAt,omitempty"`
	UpdatedAt      *string    `json:"updatedAt,omitempty"`
	RowVersion     int32      `json:"rowVersion"`
}

func toWBSDTO(r prjdb.PrjWbsElement) wbsDTO {
	return wbsDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID, ParentID: r.ParentID,
		Code: r.Code, Name: r.Name, LevelNo: r.LevelNo,
		IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type budgetDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	ProjectID      uuid.UUID `json:"projectId"`
	WBSElementID   uuid.UUID `json:"wbsElementId"`
	Version        int32     `json:"version"`
	BudgetAmount   string    `json:"budgetAmount"`
	Status         string    `json:"status"`
	IsBaseline     bool      `json:"isBaseline"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toBudgetDTO(r prjdb.PrjProjectBudget) budgetDTO {
	return budgetDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID, WBSElementID: r.WbsElementID,
		Version: r.Version, BudgetAmount: platform.FormatNumeric(r.BudgetAmount, 4),
		Status: r.Status, IsBaseline: r.IsBaseline,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type costDTO struct {
	ID                uuid.UUID `json:"id"`
	OrganizationID    uuid.UUID `json:"organizationId"`
	ProjectID         uuid.UUID `json:"projectId"`
	WBSElementID      uuid.UUID `json:"wbsElementId"`
	CostType          string    `json:"costType"`
	Amount            string    `json:"amount"`
	SourceDocumentRef *string   `json:"sourceDocumentRef,omitempty"`
	PostingDate       string    `json:"postingDate"`
	CreatedAt         *string   `json:"createdAt,omitempty"`
	UpdatedAt         *string   `json:"updatedAt,omitempty"`
	RowVersion        int32     `json:"rowVersion"`
}

func toCostDTO(r prjdb.PrjProjectCost) costDTO {
	return costDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID, WBSElementID: r.WbsElementID,
		CostType: r.CostType, Amount: platform.FormatNumeric(r.Amount, 4),
		SourceDocumentRef: r.SourceDocumentRef, PostingDate: platform.FormatDate(r.PostingDate),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type progressDTO struct {
	ID                  uuid.UUID `json:"id"`
	OrganizationID      uuid.UUID `json:"organizationId"`
	WBSElementID        uuid.UUID `json:"wbsElementId"`
	PeriodDate          string    `json:"periodDate"`
	PhysicalProgressPct string    `json:"physicalProgressPct"`
	RecordedAt          *string   `json:"recordedAt,omitempty"`
	CreatedAt           *string   `json:"createdAt,omitempty"`
	UpdatedAt           *string   `json:"updatedAt,omitempty"`
	RowVersion          int32     `json:"rowVersion"`
}

func toProgressDTO(r prjdb.PrjProjectProgress) progressDTO {
	return progressDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WBSElementID: r.WbsElementID,
		PeriodDate: platform.FormatDate(r.PeriodDate), PhysicalProgressPct: platform.FormatNumeric(r.PhysicalProgressPct, 2),
		RecordedAt: fmtOptTS(r.RecordedAt), CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type billingDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	ProjectID      uuid.UUID `json:"projectId"`
	BillingType    string    `json:"billingType"`
	Amount         string    `json:"amount"`
	ArInvoiceRef   *string   `json:"arInvoiceRef,omitempty"`
	Status         string    `json:"status"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toBillingDTO(r prjdb.PrjProjectBilling) billingDTO {
	return billingDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID,
		BillingType: r.BillingType, Amount: platform.FormatNumeric(r.Amount, 4),
		ArInvoiceRef: r.ArInvoiceRef, Status: r.Status,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type changeOrderDTO struct {
	ID                 uuid.UUID `json:"id"`
	OrganizationID     uuid.UUID `json:"organizationId"`
	ProjectID          uuid.UUID `json:"projectId"`
	Description        string    `json:"description"`
	BudgetImpactAmount string    `json:"budgetImpactAmount"`
	ScheduleImpactDays int32     `json:"scheduleImpactDays"`
	Status             string    `json:"status"`
	CreatedAt          *string   `json:"createdAt,omitempty"`
	UpdatedAt          *string   `json:"updatedAt,omitempty"`
	RowVersion         int32     `json:"rowVersion"`
}

func toChangeOrderDTO(r prjdb.PrjChangeOrder) changeOrderDTO {
	return changeOrderDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID,
		Description: r.Description, BudgetImpactAmount: platform.FormatNumeric(r.BudgetImpactAmount, 4),
		ScheduleImpactDays: r.ScheduleImpactDays, Status: r.Status,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}
