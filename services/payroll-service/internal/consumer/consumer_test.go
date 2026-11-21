package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakeEmpStore struct {
	called bool
	empID  uuid.UUID
	empNo  string
	status string
}

func (f *fakeEmpStore) ConsumeEmployeeUpdated(_ context.Context, _, _, employeeID uuid.UUID, _, employeeNo, status string) error {
	f.called = true
	f.empID = employeeID
	f.empNo = employeeNo
	f.status = status
	return nil
}

func TestPayrollConsumerEmployeeUpdated(t *testing.T) {
	store := &fakeEmpStore{}
	c := &Consumer{store: store}

	// Poison: bad org UUID
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "hr.employee.updated",
		AggregateID:    uuid.NewString(),
		OrganizationID: "bad-org",
	})
	var poison *eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison, got %v", err)
	}

	empID := uuid.New()
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "hr.employee.updated",
		AggregateID:    empID.String(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"employeeId":"` + empID.String() + `","employeeNo":"EMP-01","status":"Terminated"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.called || store.empID != empID || store.empNo != "EMP-01" || store.status != "Terminated" {
		t.Fatalf("unexpected store state: %+v", store)
	}
}
