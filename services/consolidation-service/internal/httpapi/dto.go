package httpapi

import (
	"bytes"
	"encoding/json"

	"github.com/sinergiinvestasiproperti/erp/services/consolidation-service/internal/consdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func rawJSONOrEmptyObj(b []byte) json.RawMessage {
	if len(bytes.TrimSpace(b)) == 0 {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(b)
}

func groupDTO(g consdb.ConsConsolidationGroup) map[string]any {
	return map[string]any{
		"id":                       g.ID,
		"organizationId":           g.OrganizationID,
		"code":                     g.Code,
		"name":                     g.Name,
		"parentGroupId":            g.ParentGroupID,
		"presentationCurrencyCode": g.PresentationCurrencyCode,
		"isSystem":                 g.OrganizationID == platform.SystemOrgID,
		"rowVersion":               g.RowVersion,
	}
}

func memberDTO(m consdb.ConsConsolidationMember) map[string]any {
	return map[string]any{
		"id":                   m.ID,
		"organizationId":       m.OrganizationID,
		"consolidationGroupId": m.ConsolidationGroupID,
		"companyRef":           m.CompanyRef,
		"ownershipPercentage":  platform.FormatNumeric(m.OwnershipPercentage, 4),
		"consolidationMethod":  m.ConsolidationMethod,
		"rowVersion":           m.RowVersion,
	}
}

func ruleDTO(r consdb.ConsEliminationRule) map[string]any {
	return map[string]any{
		"id":                   r.ID,
		"organizationId":       r.OrganizationID,
		"consolidationGroupId": r.ConsolidationGroupID,
		"ruleType":             r.RuleType,
		"config":               rawJSONOrEmptyObj(r.Config),
		"isActive":             r.IsActive,
		"isSystem":             r.OrganizationID == platform.SystemOrgID,
		"rowVersion":           r.RowVersion,
	}
}

func runDTO(r consdb.ConsConsolidationRun) map[string]any {
	return map[string]any{
		"id":                   r.ID,
		"organizationId":       r.OrganizationID,
		"consolidationGroupId": r.ConsolidationGroupID,
		"fiscalPeriodRef":      r.FiscalPeriodRef,
		"status":               r.Status,
		"version":              r.Version,
		"rowVersion":           r.RowVersion,
	}
}

func eliminationEntryDTO(e consdb.ConsEliminationEntry) map[string]any {
	return map[string]any{
		"id":                 e.ID,
		"organizationId":     e.OrganizationID,
		"consolidationRunId": e.ConsolidationRunID,
		"eliminationRuleId":  e.EliminationRuleID,
		"accountRef":         e.AccountRef,
		"debitAmount":        platform.FormatNumeric(e.DebitAmount, 4),
		"creditAmount":       platform.FormatNumeric(e.CreditAmount, 4),
		"description":        e.Description,
		"rowVersion":         e.RowVersion,
	}
}

func adjustmentDTO(a consdb.ConsConsolidationAdjustment) map[string]any {
	return map[string]any{
		"id":                 a.ID,
		"organizationId":     a.OrganizationID,
		"consolidationRunId": a.ConsolidationRunID,
		"accountRef":         a.AccountRef,
		"debitAmount":        platform.FormatNumeric(a.DebitAmount, 4),
		"creditAmount":       platform.FormatNumeric(a.CreditAmount, 4),
		"adjustmentType":     a.AdjustmentType,
		"notes":              a.Notes,
		"rowVersion":         a.RowVersion,
	}
}
