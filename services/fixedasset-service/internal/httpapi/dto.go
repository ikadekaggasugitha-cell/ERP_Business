package httpapi

import (
	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func categoryDTO(c fadb.FaAssetCategory) map[string]any {
	return map[string]any{
		"id":                        c.ID,
		"organizationId":            c.OrganizationID,
		"code":                      c.Code,
		"name":                      c.Name,
		"defaultDepreciationMethod": c.DefaultDepreciationMethod,
		"defaultUsefulLifeMonths":   c.DefaultUsefulLifeMonths,
		"isActive":                  c.IsActive,
		"isSystem":                  c.OrganizationID == platform.SystemOrgID,
		"rowVersion":                c.RowVersion,
	}
}

func assetDTO(a fadb.FaFixedAsset) map[string]any {
	return map[string]any{
		"id":                           a.ID,
		"organizationId":               a.OrganizationID,
		"assetNo":                      a.AssetNo,
		"name":                         a.Name,
		"assetCategoryId":              a.AssetCategoryID,
		"financialOrganizationRef":     a.FinancialOrganizationRef,
		"acquisitionDate":              platform.FormatDate(a.AcquisitionDate),
		"acquisitionCost":              platform.FormatNumeric(a.AcquisitionCost, 4),
		"salvageValue":                 platform.FormatNumeric(a.SalvageValue, 4),
		"usefulLifeMonths":             a.UsefulLifeMonths,
		"depreciationMethod":           a.DepreciationMethod,
		"depreciationStartDate":        platform.FormatDate(a.DepreciationStartDate),
		"assetAccountId":               a.AssetAccountID,
		"accumDepreciationAccountId":   a.AccumDepreciationAccountID,
		"depreciationExpenseAccountId": a.DepreciationExpenseAccountID,
		"costCenterRef":                a.CostCenterRef,
		"netBookValue":                 platform.FormatNumeric(a.NetBookValue, 4),
		"parentAssetId":                a.ParentAssetID,
		"status":                       a.Status,
		"rowVersion":                   a.RowVersion,
	}
}

func depreciationEntryDTO(e fadb.FaDepreciationEntry) map[string]any {
	return map[string]any{
		"id":                      e.ID,
		"organizationId":          e.OrganizationID,
		"fixedAssetId":            e.FixedAssetID,
		"fiscalPeriodRef":         e.FiscalPeriodRef,
		"depreciationAmount":      platform.FormatNumeric(e.DepreciationAmount, 4),
		"accumulatedDepreciation": platform.FormatNumeric(e.AccumulatedDepreciation, 4),
		"netBookValue":            platform.FormatNumeric(e.NetBookValue, 4),
		"isPosted":                e.IsPosted,
		"glJournalRef":            e.GlJournalRef,
		"runDate":                 platform.FormatDate(e.RunDate),
		"rowVersion":              e.RowVersion,
	}
}

func transactionDTO(t fadb.FaAssetTransaction) map[string]any {
	var gainLoss any
	if t.GainLossAmount.Valid {
		gainLoss = platform.FormatNumeric(t.GainLossAmount, 4)
	}
	return map[string]any{
		"id":                           t.ID,
		"organizationId":               t.OrganizationID,
		"fixedAssetId":                 t.FixedAssetID,
		"transactionType":              t.TransactionType,
		"transactionDate":              platform.FormatDate(t.TransactionDate),
		"amount":                       platform.FormatNumeric(t.Amount, 4),
		"gainLossAmount":               gainLoss,
		"fromFinancialOrganizationRef": t.FromFinancialOrganizationRef,
		"toFinancialOrganizationRef":   t.ToFinancialOrganizationRef,
		"glJournalRef":                 t.GlJournalRef,
		"notes":                        t.Notes,
		"rowVersion":                   t.RowVersion,
	}
}
