package httpapi

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/crm-service/internal/crmdb"
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

func fmtOptNumeric(n pgtype.Numeric, scale int) *string {
	if !n.Valid {
		return nil
	}
	s := platform.FormatNumeric(n, scale)
	return &s
}

func rawJSONOrEmptyObj(b []byte) json.RawMessage {
	if len(bytes.TrimSpace(b)) == 0 {
		return json.RawMessage("{}")
	}
	return json.RawMessage(b)
}

type leadDTO struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Source         *string    `json:"source,omitempty"`
	Status         string     `json:"status"`
	EstimatedValue *string    `json:"estimatedValue,omitempty"`
	CustomerRef    *uuid.UUID `json:"customerRef,omitempty"`
	CreatedAt      *string    `json:"createdAt,omitempty"`
	UpdatedAt      *string    `json:"updatedAt,omitempty"`
	RowVersion     int32      `json:"rowVersion"`
}

func toLeadDTO(r crmdb.CrmLead) leadDTO {
	return leadDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, Code: r.Code, Name: r.Name,
		Source: r.Source, Status: r.Status, EstimatedValue: fmtOptNumeric(r.EstimatedValue, 4),
		CustomerRef: r.CustomerRef, CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt),
		RowVersion: r.RowVersion,
	}
}

type opportunityDTO struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    uuid.UUID  `json:"organizationId"`
	Code              string     `json:"code"`
	LeadID            *uuid.UUID `json:"leadId,omitempty"`
	CustomerRef       *uuid.UUID `json:"customerRef,omitempty"`
	Stage             string     `json:"stage"`
	ProbabilityPct    *string    `json:"probabilityPct,omitempty"`
	EstimatedValue    *string    `json:"estimatedValue,omitempty"`
	ExpectedCloseDate *string    `json:"expectedCloseDate,omitempty"`
	Status            string     `json:"status"`
	CreatedAt         *string    `json:"createdAt,omitempty"`
	UpdatedAt         *string    `json:"updatedAt,omitempty"`
	RowVersion        int32      `json:"rowVersion"`
}

func toOpportunityDTO(r crmdb.CrmOpportunity) opportunityDTO {
	return opportunityDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, Code: r.Code, LeadID: r.LeadID,
		CustomerRef: r.CustomerRef, Stage: r.Stage,
		ProbabilityPct:    fmtOptNumeric(r.ProbabilityPct, 2),
		EstimatedValue:    fmtOptNumeric(r.EstimatedValue, 4),
		ExpectedCloseDate: fmtOptDate(r.ExpectedCloseDate), Status: r.Status,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type segmentDTO struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organizationId"`
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Criteria       json.RawMessage `json:"criteria"`
	IsSystem       bool            `json:"isSystem"`
	CreatedAt      *string         `json:"createdAt,omitempty"`
	UpdatedAt      *string         `json:"updatedAt,omitempty"`
	RowVersion     int32           `json:"rowVersion"`
}

func toSegmentDTO(r crmdb.CrmCustomerSegment) segmentDTO {
	return segmentDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, Code: r.Code, Name: r.Name,
		Criteria: rawJSONOrEmptyObj(r.Criteria), IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type interactionDTO struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organizationId"`
	CustomerRef     *uuid.UUID `json:"customerRef,omitempty"`
	LeadID          *uuid.UUID `json:"leadId,omitempty"`
	InteractionType string     `json:"interactionType"`
	OccurredAt      *string    `json:"occurredAt,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	CreatedAt       *string    `json:"createdAt,omitempty"`
	UpdatedAt       *string    `json:"updatedAt,omitempty"`
	RowVersion      int32      `json:"rowVersion"`
}

func toInteractionDTO(r crmdb.CrmInteraction) interactionDTO {
	return interactionDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, CustomerRef: r.CustomerRef, LeadID: r.LeadID,
		InteractionType: r.InteractionType, OccurredAt: fmtOptTS(r.OccurredAt), Notes: r.Notes,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type supportCaseDTO struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	CaseNo         string     `json:"caseNo"`
	CustomerRef    *uuid.UUID `json:"customerRef,omitempty"`
	Subject        string     `json:"subject"`
	Status         string     `json:"status"`
	Resolution     *string    `json:"resolution,omitempty"`
	OpenedAt       *string    `json:"openedAt,omitempty"`
	ResolvedAt     *string    `json:"resolvedAt,omitempty"`
	CreatedAt      *string    `json:"createdAt,omitempty"`
	UpdatedAt      *string    `json:"updatedAt,omitempty"`
	RowVersion     int32      `json:"rowVersion"`
}

func toSupportCaseDTO(r crmdb.CrmSupportCase) supportCaseDTO {
	return supportCaseDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, CaseNo: r.CaseNo, CustomerRef: r.CustomerRef,
		Subject: r.Subject, Status: r.Status, Resolution: r.Resolution,
		OpenedAt: fmtOptTS(r.OpenedAt), ResolvedAt: fmtOptTS(r.ResolvedAt),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type activityTaskDTO struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organizationId"`
	AssigneeRef    uuid.UUID  `json:"assigneeRef"`
	RelatedType    *string    `json:"relatedType,omitempty"`
	RelatedRef     *uuid.UUID `json:"relatedRef,omitempty"`
	DueDate        *string    `json:"dueDate,omitempty"`
	Status         string     `json:"status"`
	CreatedAt      *string    `json:"createdAt,omitempty"`
	UpdatedAt      *string    `json:"updatedAt,omitempty"`
	RowVersion     int32      `json:"rowVersion"`
}

func toActivityTaskDTO(r crmdb.CrmActivityTask) activityTaskDTO {
	return activityTaskDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, AssigneeRef: r.AssigneeRef,
		RelatedType: r.RelatedType, RelatedRef: r.RelatedRef, DueDate: fmtOptDate(r.DueDate),
		Status: r.Status, CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt),
		RowVersion: r.RowVersion,
	}
}
