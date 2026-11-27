package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/user-service/internal/userdb"
)

type Store interface {
	CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.CreateAppUserParams, ev userdb.InsertOutboxParams) (userdb.CreateAppUserRow, error)
	ProvisionAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.CreateAppUserParams, assignments []userdb.CreateUserOrgAssignmentParams, ev userdb.InsertOutboxParams) (userdb.CreateAppUserRow, []userdb.CreateUserOrgAssignmentRow, error)
	GetWithRoles(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (userdb.UsrAppUser, []userdb.ListUserOrgAssignmentsRow, error)
	GetByUsernameWithRoles(ctx context.Context, scope []uuid.UUID, username string) (userdb.UsrAppUser, []userdb.ListUserOrgAssignmentsRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg userdb.ListAppUsersParams) ([]userdb.ListAppUsersRow, int64, error)
	UpdateAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.UpdateAppUserParams, ev userdb.InsertOutboxParams) (userdb.UpdateAppUserRow, error)
	ChangeStatusAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.ChangeUserStatusParams, ev userdb.InsertOutboxParams) (userdb.ChangeUserStatusRow, error)
	DeleteUser(ctx context.Context, scope []uuid.UUID, arg userdb.DeleteAppUserParams) (int64, error)
	AssignRoleAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.CreateUserOrgAssignmentParams, ev userdb.InsertOutboxParams) (userdb.CreateUserOrgAssignmentRow, error)
	RemoveRoleAndEmit(ctx context.Context, scope []uuid.UUID, arg userdb.RemoveUserOrgAssignmentParams, ev userdb.InsertOutboxParams) (int64, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /users", s.create)
	mux.HandleFunc("POST /users/provision", s.provisionUser)
	mux.HandleFunc("GET /users", s.list)
	mux.HandleFunc("GET /users/lookup", s.getByUsername)
	mux.HandleFunc("GET /users/{id}", s.get)
	mux.HandleFunc("PUT /users/{id}", s.update)
	mux.HandleFunc("DELETE /users/{id}", s.deleteUser)
	mux.HandleFunc("POST /users/{id}/disable", s.disable)
	mux.HandleFunc("GET /users/{id}/roles", s.listRoles)
	mux.HandleFunc("POST /users/{id}/roles", s.assignRole)
	mux.HandleFunc("DELETE /users/{id}/roles/{roleId}", s.removeRole)
}

type createReq struct {
	Username              string `json:"username"`
	Email                 string `json:"email"`
	FullName              string `json:"fullName"`
	DefaultOrganizationID string `json:"defaultOrganizationId"`
}

type updateReq struct {
	Email                 string `json:"email"`
	FullName              string `json:"fullName"`
	DefaultOrganizationID string `json:"defaultOrganizationId"`
}

type assignRoleReq struct {
	RoleID            uuid.UUID `json:"roleId"`
	RoleCode          string    `json:"roleCode,omitempty"`
	OrganizationRefID string    `json:"organizationRefId"`
}

type provisionUserReq struct {
	Username              string          `json:"username"`
	Email                 string          `json:"email"`
	FullName              string          `json:"fullName"`
	DefaultOrganizationID string          `json:"defaultOrganizationId"`
	Assignments           []assignRoleReq `json:"assignments"`
}

func roleTierForCode(code string) int {
	c := strings.ToUpper(strings.TrimSpace(code))
	if c == "" {
		return httpx.AdminTierNone
	}
	return httpx.AdminTierForRoles([]string{c})
}

func orgInScope(scope []uuid.UUID, target uuid.UUID) bool {
	if target == uuid.Nil || target == platform.SystemOrgID {
		return false
	}
	for _, s := range scope {
		if s == target {
			return true
		}
	}
	return false
}

func validEmail(s string) bool {
	at := strings.IndexByte(s, '@')
	return at > 0 && at < len(s)-1 && !strings.ContainsAny(s, " \t\r\n")
}

func parseOptionalUUID(field, raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	id, err := httpx.ParseUUIDParam(field, raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func formatRoles(rows []userdb.ListUserOrgAssignmentsRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id":                r.ID,
			"userId":            r.UserID,
			"organizationRefId": r.OrganizationRefID,
			"roleId":            r.RoleID,
			"rowVersion":        r.RowVersion,
		})
	}
	return out
}
