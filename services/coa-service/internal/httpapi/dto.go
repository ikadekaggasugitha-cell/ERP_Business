package httpapi

import "encoding/json"

type chartReq struct {
	Code      string          `json:"code"`
	Name      string          `json:"name"`
	Structure json.RawMessage `json:"structure,omitempty"`
}

type accountReq struct {
	ChartOfAccountsID string `json:"chartOfAccountsId"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	AccountType       string `json:"accountType"`
	AccountClass      string `json:"accountClass"`
	IsControl         bool   `json:"isControl"`
	IsMonetary        bool   `json:"isMonetary"`
	AllowPosting      *bool  `json:"allowPosting"`
	ParentID          string `json:"parentId"`
}

type accountDTO struct {
	ID                string  `json:"id"`
	ChartOfAccountsID string  `json:"chartOfAccountsId,omitempty"`
	Code              string  `json:"code,omitempty"`
	Name              string  `json:"name,omitempty"`
	AccountType       string  `json:"accountType,omitempty"`
	AccountClass      *string `json:"accountClass,omitempty"`
	IsControl         bool    `json:"isControl"`
	IsMonetary        bool    `json:"isMonetary"`
	AllowPosting      bool    `json:"allowPosting"`
	RowVersion        int32   `json:"rowVersion"`
}

type finOrgReq struct {
	Code                   string `json:"code"`
	Name                   string `json:"name"`
	OrganizationRefID      string `json:"organizationRefId"`
	FunctionalCurrencyCode string `json:"functionalCurrencyCode"`
	ChartOfAccountsID      string `json:"chartOfAccountsId"`
	FiscalCalendarID       string `json:"fiscalCalendarId"`
}

type ledgerReq struct {
	FinancialOrganizationID string `json:"financialOrganizationId"`
	Code                    string `json:"code"`
	LedgerType              string `json:"ledgerType"`
	CurrencyCode            string `json:"currencyCode"`
}

type calendarReq struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	PeriodType string `json:"periodType"`
}

type yearReq struct {
	FiscalCalendarID string `json:"fiscalCalendarId"`
	YearNo           int32  `json:"yearNo"`
	Name             string `json:"name"`
	StartDate        string `json:"startDate"`
	EndDate          string `json:"endDate"`
}

type periodReq struct {
	FiscalYearID string `json:"fiscalYearId"`
	PeriodNo     int32  `json:"periodNo"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
}

type accountTreeNode struct {
	ID              string             `json:"id"`
	Code            string             `json:"code"`
	Name            string             `json:"name"`
	AccountType     string             `json:"accountType"`
	ParentAccountID string             `json:"parentAccountId,omitempty"`
	AllowPosting    bool               `json:"allowPosting"`
	RowVersion      int32              `json:"rowVersion"`
	Children        []*accountTreeNode `json:"children"`
}
