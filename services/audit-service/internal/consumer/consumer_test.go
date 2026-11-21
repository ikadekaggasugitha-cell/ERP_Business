package consumer

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/audit-service/internal/auditdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type stubRecorder struct {
	called bool
	scope  []uuid.UUID
	proc   auditdb.MarkEventProcessedParams
	entry  auditdb.RecordAuditLogParams
}

func (s *stubRecorder) RecordFromEvent(_ context.Context, scope []uuid.UUID, proc auditdb.MarkEventProcessedParams, entry auditdb.RecordAuditLogParams) (bool, error) {
	s.called = true
	s.scope = scope
	s.proc = proc
	s.entry = entry
	return false, nil
}

func TestHandleRoutesPoisonAndPersistsValidEvents(t *testing.T) {
	rec := &stubRecorder{}
	c := NewHandler(rec)

	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        "not-a-uuid",
		EventType:      "gl.journal.posted",
		OrganizationID: uuid.NewString(),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for invalid eventId, got %v", err)
	}
	if rec.called {
		t.Fatal("repo must not be called on poison message")
	}

	evID := uuid.New()
	orgID := uuid.New()
	aggID := uuid.New()
	corrID := uuid.New()
	actorID := uuid.New()
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        evID.String(),
		EventType:      "gl.journal.posted",
		AggregateType:  "JournalEntry",
		AggregateID:    aggID.String(),
		CorrelationID:  corrID.String(),
		OrganizationID: orgID.String(),
		Data:           []byte(`{"journalNumber":"JV-1","actorId":"` + actorID.String() + `"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rec.called || rec.proc.EventID != evID || rec.entry.Action != "gl.journal.posted" {
		t.Fatalf("unexpected recorded args: proc=%+v entry=%+v", rec.proc, rec.entry)
	}
	if rec.entry.ActorID == nil || *rec.entry.ActorID != actorID {
		t.Fatalf("expected ActorID=%s, got %v", actorID, rec.entry.ActorID)
	}
}

func TestHandlePersistsCurrencyEvents(t *testing.T) {
	tenantOrgID := uuid.New()
	cases := []struct {
		name      string
		eventType string
		aggType   string
		orgID     uuid.UUID
		payload   string
	}{
		{
			name:      "currency.created/tenant_org",
			eventType: "currency.created",
			aggType:   "Currency",
			orgID:     tenantOrgID,
			payload:   `{"code":"SGD","name":"Singapore Dollar","precision":2}`,
		},
		{
			name:      "currency.created/system_org",
			eventType: "currency.created",
			aggType:   "Currency",
			orgID:     uuid.Nil,
			payload:   `{"code":"JPY","name":"Japanese Yen","precision":0}`,
		},
		{
			name:      "currency.rate.created/system_org",
			eventType: "currency.rate.created",
			aggType:   "ExchangeRate",
			orgID:     uuid.Nil,
			payload:   `{"fromCurrency":"USD","toCurrency":"IDR","rate":"15850.00000000"}`,
		},
		{
			name:      "currency.rate.updated/tenant_org",
			eventType: "currency.rate.updated",
			aggType:   "ExchangeRate",
			orgID:     tenantOrgID,
			payload:   `{"rate":"15900.00000000"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &stubRecorder{}
			c := NewHandler(rec)
			evID := uuid.New()
			aggID := uuid.New()

			err := c.Handle(context.Background(), eventbus.Envelope{
				EventID:        evID.String(),
				EventType:      tc.eventType,
				AggregateType:  tc.aggType,
				AggregateID:    aggID.String(),
				OrganizationID: tc.orgID.String(),
				Data:           []byte(tc.payload),
			})
			if err != nil {
				t.Fatalf("Handle(%s): %v", tc.eventType, err)
			}
			if !rec.called {
				t.Fatalf("expected recorder to be called for %s", tc.eventType)
			}
			if len(rec.scope) != 1 || rec.scope[0] != tc.orgID {
				t.Fatalf("scope = %v, want [%s]", rec.scope, tc.orgID)
			}
			if rec.proc.EventID != evID || rec.proc.EventType != tc.eventType {
				t.Fatalf("unexpected processed params: %+v", rec.proc)
			}
			if rec.entry.Action != tc.eventType || rec.entry.EntityType != tc.aggType || rec.entry.OrganizationID != tc.orgID {
				t.Fatalf("unexpected audit entry: %+v", rec.entry)
			}
			if rec.entry.EntityID == nil || *rec.entry.EntityID != aggID {
				t.Fatalf("unexpected entityID: %v, want %s", rec.entry.EntityID, aggID)
			}
			if string(rec.entry.AfterValue) != tc.payload {
				t.Fatalf("AfterValue = %s, want %s", rec.entry.AfterValue, tc.payload)
			}
		})
	}
}

func TestHandleRejectsPoisonInputsAndTenantNilOrg(t *testing.T) {
	validUUID := uuid.NewString()
	cases := []struct {
		name string
		env  eventbus.Envelope
	}{
		{
			name: "nil eventId",
			env: eventbus.Envelope{
				EventID:        uuid.Nil.String(),
				EventType:      "currency.created",
				OrganizationID: uuid.Nil.String(),
			},
		},
		{
			name: "empty eventType",
			env: eventbus.Envelope{
				EventID:        validUUID,
				EventType:      "   ",
				OrganizationID: validUUID,
			},
		},
		{
			name: "empty organizationId",
			env: eventbus.Envelope{
				EventID:        validUUID,
				EventType:      "currency.created",
				OrganizationID: "",
			},
		},
		{
			name: "malformed organizationId",
			env: eventbus.Envelope{
				EventID:        validUUID,
				EventType:      "currency.created",
				OrganizationID: "not-a-uuid",
			},
		},
		{
			name: "nil organizationId on non-currency tenant event",
			env: eventbus.Envelope{
				EventID:        validUUID,
				EventType:      "gl.journal.posted",
				OrganizationID: uuid.Nil.String(),
			},
		},
		{
			name: "invalid correlationId",
			env: eventbus.Envelope{
				EventID:        validUUID,
				EventType:      "gl.journal.posted",
				OrganizationID: validUUID,
				CorrelationID:  "bad-corr",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := &stubRecorder{}
			c := NewHandler(rec)
			err := c.Handle(context.Background(), tc.env)
			var poison eventbus.ErrPoison
			if !errors.As(err, &poison) {
				t.Fatalf("expected ErrPoison, got %v", err)
			}
			if rec.called {
				t.Fatal("repo must not be called on poison message")
			}
		})
	}
}

func TestAllTopicsContainsExpectedDomainTopics(t *testing.T) {
	required := []string{
		eventbus.TopicFor("gl.journal.posted"),
		eventbus.TopicFor("gl.journal.reversed"),
		eventbus.TopicFor("coa.account.upserted"),
		eventbus.TopicFor("coa.period.updated"),
		eventbus.TopicFor("ap.invoice.recorded"),
		eventbus.TopicFor("ar.invoice.issued"),
		eventbus.TopicFor("ar.invoice.paid"),
		eventbus.TopicFor("tax.invoice.issued"),
		eventbus.TopicFor("tax.efaktur.reported"),
		eventbus.TopicFor("treasury.payment.settled"),
		eventbus.TopicFor("treasury.payment.reconciled"),
		eventbus.TopicFor("cashbank.payment.settled"),
		eventbus.TopicFor("hr.employee.updated"),
		eventbus.TopicFor("payroll.run.calculated"),
		eventbus.TopicFor("payroll.run.completed"),
		eventbus.TopicFor("payroll.run.approved"),
		eventbus.TopicFor("payroll.run.posted"),
		eventbus.TopicFor("procurement.po.approved"),
		eventbus.TopicFor("procurement.goods.received"),
		eventbus.TopicFor("inventory.stock.reserved"),
		eventbus.TopicFor("inventory.goods.issued"),
		eventbus.TopicFor("inventory.reservation.failed"),
		eventbus.TopicFor("sales.order.confirmed"),
		eventbus.TopicFor("sales.delivery.completed"),
		eventbus.TopicFor("manufacturing.workorder.released"),
		eventbus.TopicFor("manufacturing.workorder.completed"),
		eventbus.TopicFor("masterdata.party.upserted"),
		eventbus.TopicFor("masterdata.item.upserted"),
		eventbus.TopicFor("org.unit.upserted"),
		eventbus.TopicFor("security.role.upserted"),
		eventbus.TopicFor("user.account.upserted"),
		eventbus.TopicFor("workflow.approval.requested"),
		eventbus.TopicFor("workflow.approval.completed"),
		eventbus.TopicFor("currency.created"),
		eventbus.TopicFor("currency.rate.created"),
		eventbus.TopicFor("currency.rate.updated"),
		eventbus.TopicFor("dimension.cost_center.upserted"),
		eventbus.TopicFor("dimension.profit_center.upserted"),
		eventbus.TopicFor("dimension.allocation_rule.upserted"),
		eventbus.TopicFor("dimension.allocation_rule.executed"),
		eventbus.TopicFor("localization.language.updated"),
		eventbus.TopicFor("localization.bundle.updated"),
		eventbus.TopicFor("localization.locale_format.updated"),
		eventbus.TopicFor("fixedasset.asset.acquired"),
		eventbus.TopicFor("fixedasset.depreciation.posted"),
		eventbus.TopicFor("fixedasset.asset.disposed"),
		eventbus.TopicFor("budget.approved"),
		eventbus.TopicFor("budget.commitment.created"),
		eventbus.TopicFor("financialclose.period.closed"),
		eventbus.TopicFor("financialclose.period.locked"),
		eventbus.TopicFor("financialclose.period.reopened"),
		eventbus.TopicFor("consolidation.run.finalized"),
		eventbus.TopicFor("project.billing.approved"),
		eventbus.TopicFor("asset.movement.completed"),
		eventbus.TopicFor("asset.verification.recorded"),
		eventbus.TopicFor("maintenance.workorder.parts_requested"),
		eventbus.TopicFor("maintenance.workorder.completed"),
		eventbus.TopicFor("crm.opportunity.won"),
	}

	seen := make(map[string]bool, len(AllTopics))
	for _, topic := range AllTopics {
		if seen[topic] {
			t.Fatalf("duplicate topic %q in AllTopics", topic)
		}
		seen[topic] = true
	}
	for _, want := range required {
		if !seen[want] {
			t.Fatalf("expected AllTopics to contain %q", want)
		}
	}
	if len(AllTopics) != len(required) {
		t.Fatalf("AllTopics len=%d, want %d", len(AllTopics), len(required))
	}
}

func TestAllTopicsParityWithGatewayDomainTopics(t *testing.T) {
	hubPath := filepath.Join("..", "..", "..", "api-gateway", "internal", "sse", "hub.go")
	if _, err := os.Stat(hubPath); os.IsNotExist(err) {
		t.Skipf("sibling api-gateway file not present at %s", hubPath)
	} else if err != nil {
		t.Fatalf("stat %s: %v", hubPath, err)
	}

	domainTopics := parseStringSliceVar(t, hubPath, "DomainTopics")
	auditSet := make(map[string]bool, len(AllTopics))
	for _, topic := range AllTopics {
		auditSet[topic] = true
	}
	domainSet := make(map[string]bool, len(domainTopics))
	for _, topic := range domainTopics {
		if domainSet[topic] {
			t.Fatalf("duplicate topic %q in api-gateway DomainTopics", topic)
		}
		domainSet[topic] = true
		if !auditSet[topic] {
			t.Fatalf("api-gateway DomainTopics has %q missing from audit-service AllTopics", topic)
		}
	}
	for _, topic := range AllTopics {
		if !domainSet[topic] {
			t.Fatalf("audit-service AllTopics has %q missing from api-gateway DomainTopics", topic)
		}
	}
}

func parseStringSliceVar(t *testing.T, filePath, varName string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filePath, err)
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || vs.Names[0].Name != varName || len(vs.Values) == 0 {
				continue
			}
			lit, ok := vs.Values[0].(*ast.CompositeLit)
			if !ok {
				t.Fatalf("%s: %s is not a composite literal", filePath, varName)
			}
			out := make([]string, 0, len(lit.Elts))
			for _, elt := range lit.Elts {
				switch node := elt.(type) {
				case *ast.BasicLit:
					if node.Kind != token.STRING {
						t.Fatalf("%s: non-string literal in %s", filePath, varName)
					}
					v, err := strconv.Unquote(node.Value)
					if err != nil {
						t.Fatalf("%s: unquote %s: %v", filePath, node.Value, err)
					}
					out = append(out, v)
				case *ast.CallExpr:
					sel, ok := node.Fun.(*ast.SelectorExpr)
					if !ok {
						t.Fatalf("%s: unsupported call expr in %s", filePath, varName)
					}
					pkg, pkgOK := sel.X.(*ast.Ident)
					if !pkgOK || pkg.Name != "eventbus" || sel.Sel.Name != "TopicFor" || len(node.Args) != 1 {
						t.Fatalf("%s: unsupported call expr in %s", filePath, varName)
					}
					arg, ok := node.Args[0].(*ast.BasicLit)
					if !ok || arg.Kind != token.STRING {
						t.Fatalf("%s: unsupported TopicFor arg in %s", filePath, varName)
					}
					v, err := strconv.Unquote(arg.Value)
					if err != nil {
						t.Fatalf("%s: unquote %s: %v", filePath, arg.Value, err)
					}
					out = append(out, eventbus.TopicFor(v))
				default:
					t.Fatalf("%s: unsupported AST element %T in %s", filePath, elt, varName)
				}
			}
			return out
		}
	}
	t.Fatalf("%s: var %s not found", filePath, varName)
	return nil
}

