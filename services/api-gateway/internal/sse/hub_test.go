package sse

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestHubBroadcastFiltersByOrg(t *testing.T) {
	hub := NewHub()
	subA, cancelA := hub.Subscribe("11111111-1111-1111-1111-111111111111")
	defer cancelA()
	subB, cancelB := hub.Subscribe("22222222-2222-2222-2222-222222222222")
	defer cancelB()

	ev := StreamEvent{
		EventID:        "evt-1",
		EventType:      "gl.journal.posted",
		OrganizationID: "11111111-1111-1111-1111-111111111111",
		Payload:        json.RawMessage(`{"journalNo":"JV-1"}`),
	}
	hub.Broadcast(ev)

	select {
	case got := <-subA.Ch:
		if got.EventID != "evt-1" {
			t.Fatalf("unexpected event: %+v", got)
		}
	default:
		t.Fatal("expected event delivered to matching org subscriber A")
	}

	select {
	case got := <-subB.Ch:
		t.Fatalf("did not expect cross-tenant event in subscriber B: %+v", got)
	default:
	}
}

func TestHubBroadcastDeliversSystemOrgEventsToTenantSubscribers(t *testing.T) {
	hub := NewHub()
	subA, cancelA := hub.Subscribe("11111111-1111-1111-1111-111111111111")
	defer cancelA()
	subB, cancelB := hub.Subscribe("22222222-2222-2222-2222-222222222222")
	defer cancelB()

	ev := StreamEvent{
		EventID:        "evt-sys-cur-1",
		EventType:      "currency.created",
		OrganizationID: SystemOrgID,
		Payload:        json.RawMessage(`{"code":"JPY"}`),
	}
	hub.Broadcast(ev)

	for name, ch := range map[string]<-chan StreamEvent{"subA": subA.Ch, "subB": subB.Ch} {
		select {
		case got := <-ch:
			if got.EventID != "evt-sys-cur-1" {
				t.Fatalf("%s received unexpected event: %+v", name, got)
			}
		default:
			t.Fatalf("expected system-org currency event delivered to %s", name)
		}
	}
}

func TestDomainTopicsIncludesSynchronizedTopics(t *testing.T) {
	required := []string{
		"erp.gl.journal.posted.v1",
		"erp.gl.journal.reversed.v1",
		"erp.coa.account.upserted.v1",
		"erp.coa.period.updated.v1",
		"erp.ap.invoice.recorded.v1",
		"erp.ar.invoice.issued.v1",
		"erp.ar.invoice.paid.v1",
		"erp.tax.invoice.issued.v1",
		"erp.tax.efaktur.reported.v1",
		"erp.treasury.payment.settled.v1",
		"erp.treasury.payment.reconciled.v1",
		"erp.cashbank.payment.settled.v1",
		"erp.hr.employee.updated.v1",
		"erp.payroll.run.calculated.v1",
		"erp.payroll.run.completed.v1",
		"erp.payroll.run.approved.v1",
		"erp.payroll.run.posted.v1",
		"erp.procurement.po.approved.v1",
		"erp.procurement.goods.received.v1",
		"erp.inventory.stock.reserved.v1",
		"erp.inventory.goods.issued.v1",
		"erp.inventory.reservation.failed.v1",
		"erp.sales.order.confirmed.v1",
		"erp.sales.delivery.completed.v1",
		"erp.manufacturing.workorder.released.v1",
		"erp.manufacturing.workorder.completed.v1",
		"erp.masterdata.party.upserted.v1",
		"erp.masterdata.item.upserted.v1",
		"erp.org.unit.upserted.v1",
		"erp.security.role.upserted.v1",
		"erp.user.account.upserted.v1",
		"erp.workflow.approval.requested.v1",
		"erp.workflow.approval.completed.v1",
		"erp.currency.created.v1",
		"erp.currency.rate.created.v1",
		"erp.currency.rate.updated.v1",
		"erp.dimension.cost_center.upserted.v1",
		"erp.dimension.profit_center.upserted.v1",
		"erp.dimension.allocation_rule.upserted.v1",
		"erp.dimension.allocation_rule.executed.v1",
		"erp.localization.language.updated.v1",
		"erp.localization.bundle.updated.v1",
		"erp.localization.locale_format.updated.v1",
		"erp.fixedasset.asset.acquired.v1",
		"erp.fixedasset.depreciation.posted.v1",
		"erp.fixedasset.asset.disposed.v1",
		"erp.budget.approved.v1",
		"erp.budget.commitment.created.v1",
		"erp.financialclose.period.closed.v1",
		"erp.financialclose.period.locked.v1",
		"erp.financialclose.period.reopened.v1",
		"erp.consolidation.run.finalized.v1",
		"erp.project.billing.approved.v1",
		"erp.asset.movement.completed.v1",
		"erp.asset.verification.recorded.v1",
		"erp.maintenance.workorder.parts_requested.v1",
		"erp.maintenance.workorder.completed.v1",
		"erp.crm.opportunity.won.v1",
	}
	topicSet := make(map[string]bool, len(DomainTopics))
	for _, topic := range DomainTopics {
		topicSet[topic] = true
	}
	for _, want := range required {
		if !topicSet[want] {
			t.Fatalf("expected DomainTopics to contain %q", want)
		}
	}
	if topicSet["erp.masterdata.party.updated.v1"] {
		t.Fatal("expected deprecated erp.masterdata.party.updated.v1 to be replaced by erp.masterdata.party.upserted.v1")
	}
	if len(topicSet) != len(DomainTopics) {
		t.Fatalf("DomainTopics contains duplicates: len=%d unique=%d", len(DomainTopics), len(topicSet))
	}
	if len(DomainTopics) != len(required) {
		t.Fatalf("DomainTopics len=%d, want %d", len(DomainTopics), len(required))
	}
}

func TestDomainTopicsParityWithAuditAllTopics(t *testing.T) {
	consumerPath := filepath.Join("..", "..", "..", "audit-service", "internal", "consumer", "consumer.go")
	if _, err := os.Stat(consumerPath); os.IsNotExist(err) {
		t.Skipf("sibling audit-service file not present at %s", consumerPath)
	} else if err != nil {
		t.Fatalf("stat %s: %v", consumerPath, err)
	}

	auditTopics := parseStringSliceVar(t, consumerPath, "AllTopics")
	domainSet := make(map[string]bool, len(DomainTopics))
	for _, topic := range DomainTopics {
		domainSet[topic] = true
	}
	auditSet := make(map[string]bool, len(auditTopics))
	for _, topic := range auditTopics {
		if auditSet[topic] {
			t.Fatalf("duplicate topic %q in audit-service AllTopics", topic)
		}
		auditSet[topic] = true
		if !domainSet[topic] {
			t.Fatalf("audit-service AllTopics has %q missing from api-gateway DomainTopics", topic)
		}
	}
	for _, topic := range DomainTopics {
		if !auditSet[topic] {
			t.Fatalf("api-gateway DomainTopics has %q missing from audit-service AllTopics", topic)
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
					out = append(out, "erp."+v+".v1")
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


