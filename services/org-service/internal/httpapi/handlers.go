package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/org-service/internal/orgdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	CreateOrganizationType(ctx context.Context, scope []uuid.UUID, arg orgdb.CreateOrganizationTypeParams) (orgdb.CreateOrganizationTypeRow, error)
	ListOrganizationTypesWithCount(ctx context.Context, scope []uuid.UUID, arg orgdb.ListOrganizationTypesParams) ([]orgdb.ListOrganizationTypesRow, int64, error)
	GetOrganizationTypeByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (orgdb.GetOrganizationTypeByIDRow, error)
	CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.CreateOrganizationParams, ev orgdb.InsertOutboxParams) (orgdb.CreateOrganizationRow, error)
	GetWithHierarchy(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (orgdb.OrgOrganization, []orgdb.ListOrganizationDescendantsRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg orgdb.ListOrganizationsParams) ([]orgdb.ListOrganizationsRow, int64, error)
	ListAttributes(ctx context.Context, scope []uuid.UUID, orgRefID uuid.UUID) (map[string]any, error)
	UpsertAttributes(ctx context.Context, scope []uuid.UUID, orgID, orgRefID uuid.UUID, attrs map[string]any, actor uuid.UUID) error
	UpdateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.UpdateOrganizationParams, ev orgdb.InsertOutboxParams) (orgdb.UpdateOrganizationRow, error)
	ActivateAndEmit(ctx context.Context, scope []uuid.UUID, arg orgdb.ActivateOrganizationParams, ev orgdb.InsertOutboxParams) (int32, error)
	DeleteOrganization(ctx context.Context, scope []uuid.UUID, arg orgdb.DeleteOrganizationParams) (int64, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /organization-types", s.createType)
	mux.HandleFunc("GET /organization-types", s.listTypes)

	mux.HandleFunc("POST /organizations", s.create)
	mux.HandleFunc("GET /organizations", s.list)
	mux.HandleFunc("GET /organizations/{id}", s.get)
	mux.HandleFunc("PUT /organizations/{id}", s.update)
	mux.HandleFunc("POST /organizations/{id}/activate", s.activate)
	mux.HandleFunc("DELETE /organizations/{id}", s.deleteOrg)
	mux.HandleFunc("GET /organizations/{id}/hierarchy", s.hierarchy)
	mux.HandleFunc("GET /organizations/{id}/descendants", s.descendants)
	mux.HandleFunc("GET /organizations/{id}/attributes", s.getAttributes)
	mux.HandleFunc("PUT /organizations/{id}/attributes", s.putAttributes)

	mux.HandleFunc("GET /org-units", s.list)
	mux.HandleFunc("GET /org-units/{id}", s.get)
	mux.HandleFunc("GET /org-units/{id}/hierarchy", s.hierarchy)
}

type typeReq struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	LevelHint *int32 `json:"levelHint"`
}

type orgReq struct {
	Code                    string         `json:"code"`
	Name                    string         `json:"name"`
	OrganizationTypeID      string         `json:"organizationTypeId"`
	ParentID                string         `json:"parentId"`
	EffectiveFrom           string         `json:"effectiveFrom"`
	EffectiveTo             string         `json:"effectiveTo"`
	Attributes              map[string]any `json:"attributes,omitempty"`
	AutoProvisionWarehouse  bool           `json:"autoProvisionWarehouse,omitempty"`
	AutoProvisionCostCenter bool           `json:"autoProvisionCostCenter,omitempty"`
}

func canTierCreateOrgType(callerTier int, typeCode string) bool {
	if callerTier <= httpx.AdminTierHolding {
		return true
	}
	tc := strings.ToUpper(strings.TrimSpace(typeCode))
	if callerTier == httpx.AdminTierSubsidiary {
		return tc != "HOLDING" && tc != "SUB_HOLDING"
	}
	if callerTier == httpx.AdminTierBranch {
		switch tc {
		case "HOLDING", "SUB_HOLDING", "COMPANY", "BRANCH":
			return false
		default:
			return true
		}
	}
	return false
}

func formatOptionalDate(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := platform.FormatDate(d)
	return &s
}
