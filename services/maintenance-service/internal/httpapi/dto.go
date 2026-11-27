package httpapi

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/maintenance-service/internal/mntdb"
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

type scheduleDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	AssetRef       uuid.UUID `json:"assetRef"`
	ScheduleType   string    `json:"scheduleType"`
	IntervalValue  int32     `json:"intervalValue"`
	IntervalUnit   string    `json:"intervalUnit"`
	NextDueDate    *string   `json:"nextDueDate,omitempty"`
	IsActive       bool      `json:"isActive"`
	IsSystem       bool      `json:"isSystem"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toScheduleDTO(r mntdb.MntMaintenanceSchedule) scheduleDTO {
	return scheduleDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, AssetRef: r.AssetRef,
		ScheduleType: r.ScheduleType, IntervalValue: r.IntervalValue, IntervalUnit: r.IntervalUnit,
		NextDueDate: fmtOptDate(r.NextDueDate), IsActive: r.IsActive,
		IsSystem: r.OrganizationID == platform.SystemOrgID,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type workOrderDTO struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organizationId"`
	WoNo            string     `json:"woNo"`
	AssetRef        uuid.UUID  `json:"assetRef"`
	MaintenanceType string     `json:"maintenanceType"`
	Status          string     `json:"status"`
	AssigneeRef     *uuid.UUID `json:"assigneeRef,omitempty"`
	VendorRef       *uuid.UUID `json:"vendorRef,omitempty"`
	ScheduledDate   *string    `json:"scheduledDate,omitempty"`
	DowntimeStart   *string    `json:"downtimeStart,omitempty"`
	DowntimeEnd     *string    `json:"downtimeEnd,omitempty"`
	CreatedAt       *string    `json:"createdAt,omitempty"`
	UpdatedAt       *string    `json:"updatedAt,omitempty"`
	RowVersion      int32      `json:"rowVersion"`
}

func toWorkOrderDTO(r mntdb.MntWorkOrder) workOrderDTO {
	return workOrderDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WoNo: r.WoNo, AssetRef: r.AssetRef,
		MaintenanceType: r.MaintenanceType, Status: r.Status,
		AssigneeRef: r.AssigneeRef, VendorRef: r.VendorRef,
		ScheduledDate: fmtOptDate(r.ScheduledDate),
		DowntimeStart: fmtOptTS(r.DowntimeStart), DowntimeEnd: fmtOptTS(r.DowntimeEnd),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type partDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	WorkOrderID    uuid.UUID `json:"workOrderId"`
	ItemRef        uuid.UUID `json:"itemRef"`
	Quantity       string    `json:"quantity"`
	UnitCost       string    `json:"unitCost"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toPartDTO(r mntdb.MntWorkOrderPart) partDTO {
	return partDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WorkOrderID: r.WorkOrderID,
		ItemRef: r.ItemRef, Quantity: platform.FormatNumeric(r.Quantity, 4),
		UnitCost: platform.FormatNumeric(r.UnitCost, 4),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type checklistDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	WorkOrderID    uuid.UUID `json:"workOrderId"`
	StepNo         int32     `json:"stepNo"`
	Description    string    `json:"description"`
	IsDone         bool      `json:"isDone"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toChecklistDTO(r mntdb.MntWorkOrderChecklist) checklistDTO {
	return checklistDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WorkOrderID: r.WorkOrderID,
		StepNo: r.StepNo, Description: r.Description, IsDone: r.IsDone,
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

type costDTO struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	WorkOrderID    uuid.UUID `json:"workOrderId"`
	CostType       string    `json:"costType"`
	Amount         string    `json:"amount"`
	CreatedAt      *string   `json:"createdAt,omitempty"`
	UpdatedAt      *string   `json:"updatedAt,omitempty"`
	RowVersion     int32     `json:"rowVersion"`
}

func toCostDTO(r mntdb.MntMaintenanceCost) costDTO {
	return costDTO{
		ID: r.ID, OrganizationID: r.OrganizationID, WorkOrderID: r.WorkOrderID,
		CostType: r.CostType, Amount: platform.FormatNumeric(r.Amount, 4),
		CreatedAt: fmtOptTS(r.CreatedAt), UpdatedAt: fmtOptTS(r.UpdatedAt), RowVersion: r.RowVersion,
	}
}

