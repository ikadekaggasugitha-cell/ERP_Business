package httpapi

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/dimension-service/internal/dimensiondb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type centerReq struct {
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	ParentID      *uuid.UUID `json:"parentId,omitempty"`
	OwnerRef      *uuid.UUID `json:"ownerRef,omitempty"`
	Status        string     `json:"status,omitempty"`
	EffectiveFrom string     `json:"effectiveFrom"`
	EffectiveTo   string     `json:"effectiveTo,omitempty"`
}

type centerResp struct {
	ID            uuid.UUID  `json:"id"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	ParentID      *uuid.UUID `json:"parentId,omitempty"`
	OwnerRef      *uuid.UUID `json:"ownerRef,omitempty"`
	Status        string     `json:"status"`
	EffectiveFrom string     `json:"effectiveFrom"`
	EffectiveTo   string     `json:"effectiveTo,omitempty"`
	RowVersion    int32      `json:"rowVersion"`
}

func costCenterDTO(c dimensiondb.CcCostCenter) centerResp {
	return centerResp{
		ID: c.ID, Code: c.Code, Name: c.Name, Description: c.Description, ParentID: c.ParentID,
		OwnerRef: c.OwnerRef, Status: c.Status, EffectiveFrom: platform.FormatDate(c.EffectiveFrom),
		EffectiveTo: platform.FormatDate(c.EffectiveTo), RowVersion: c.RowVersion,
	}
}

func profitCenterDTO(c dimensiondb.PcProfitCenter) centerResp {
	return centerResp{
		ID: c.ID, Code: c.Code, Name: c.Name, ParentID: c.ParentID, Status: c.Status,
		EffectiveFrom: platform.FormatDate(c.EffectiveFrom), EffectiveTo: platform.FormatDate(c.EffectiveTo),
		RowVersion: c.RowVersion,
	}
}

type allocTarget struct {
	CostCenterID uuid.UUID `json:"costCenterId"`
	Percentage   string    `json:"percentage"`
}

type allocReq struct {
	SourceCostCenterID uuid.UUID     `json:"sourceCostCenterId"`
	Name               string        `json:"name"`
	AllocationBasis    string        `json:"allocationBasis"`
	Targets            []allocTarget `json:"targets"`
	IsActive           *bool         `json:"isActive,omitempty"`
}

type allocResp struct {
	ID                 uuid.UUID       `json:"id"`
	SourceCostCenterID uuid.UUID       `json:"sourceCostCenterId"`
	Name               string          `json:"name"`
	AllocationBasis    string          `json:"allocationBasis"`
	Targets            json.RawMessage `json:"targets"`
	IsActive           bool            `json:"isActive"`
	RowVersion         int32           `json:"rowVersion"`
}

func allocDTO(a dimensiondb.CcCostAllocationRule) allocResp {
	return allocResp{
		ID: a.ID, SourceCostCenterID: a.SourceCostCenterID, Name: a.Name, AllocationBasis: a.AllocationBasis,
		Targets: json.RawMessage(a.Targets), IsActive: a.IsActive, RowVersion: a.RowVersion,
	}
}
