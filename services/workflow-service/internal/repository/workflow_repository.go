// Package repository owns workflow_db transactions (definitions, steps,
// instances, tasks with SoD enforcement, outbox).
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/workflow-service/internal/workflowdb"
)

var (
	ErrSoDViolation   = errors.New("segregation of duties violation: requester cannot approve own request")
	ErrAlreadyDecided = errors.New("workflow task or instance already decided")
	ErrVersionConflict = errors.New("workflow task version conflict")
)

type WorkflowRepository struct {
	db *platform.DB
	q  *workflowdb.Queries
}

func NewWorkflowRepository(db *platform.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db, q: workflowdb.New(db.Pool)}
}

func (r *WorkflowRepository) CreateDefinition(ctx context.Context, scope []uuid.UUID, arg workflowdb.CreateWorkflowDefinitionParams, steps []workflowdb.CreateWorkflowStepParams) (workflowdb.CreateWorkflowDefinitionRow, error) {
	var out workflowdb.CreateWorkflowDefinitionRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.CreateWorkflowDefinition(ctx, arg); e != nil {
			return e
		}
		for _, s := range steps {
			s.OrganizationID = arg.OrganizationID
			s.DefinitionID = out.ID
			s.CreatedBy = arg.CreatedBy
			if e = q.CreateWorkflowStep(ctx, s); e != nil {
				return e
			}
		}
		return nil
	})
	return out, err
}

func (r *WorkflowRepository) ListDefinitionsWithCount(ctx context.Context, scope []uuid.UUID, arg workflowdb.ListWorkflowDefinitionsParams) ([]workflowdb.ListWorkflowDefinitionsRow, int64, error) {
	var rows []workflowdb.ListWorkflowDefinitionsRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListWorkflowDefinitions(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountWorkflowDefinitions(ctx, arg.DocumentType)
		return e
	})
	return rows, total, err
}

func (r *WorkflowRepository) CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg workflowdb.CreateWorkflowInstanceParams, assigneeID uuid.UUID, ev workflowdb.InsertOutboxParams) (workflowdb.CreateWorkflowInstanceRow, uuid.UUID, error) {
	var created workflowdb.CreateWorkflowInstanceRow
	var taskID uuid.UUID
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateWorkflowInstance(ctx, arg); e != nil {
			return e
		}
		taskRow, e := q.CreateWorkflowTask(ctx, workflowdb.CreateWorkflowTaskParams{
			OrganizationID: arg.OrganizationID,
			InstanceID:     created.ID,
			AssigneeID:     assigneeID,
			CreatedBy:      arg.CreatedBy,
		})
		if e != nil {
			return e
		}
		taskID = taskRow.ID
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, taskID, err
}

func (r *WorkflowRepository) GetWithTasks(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (workflowdb.WfWorkflowInstance, []workflowdb.ListWorkflowTasksRow, error) {
	var inst workflowdb.WfWorkflowInstance
	var tasks []workflowdb.ListWorkflowTasksRow
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if inst, e = q.GetWorkflowInstance(ctx, id); e != nil {
			return e
		}
		tasks, e = q.ListWorkflowTasks(ctx, workflowdb.ListWorkflowTasksParams{
			Limit:      50,
			Offset:     0,
			InstanceID: &id,
		})
		return e
	})
	return inst, tasks, err
}

func (r *WorkflowRepository) ListWithCount(ctx context.Context, scope []uuid.UUID, arg workflowdb.ListWorkflowInstancesParams) ([]workflowdb.ListWorkflowInstancesRow, int64, error) {
	var rows []workflowdb.ListWorkflowInstancesRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListWorkflowInstances(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountWorkflowInstances(ctx, arg.Status)
		return e
	})
	return rows, total, err
}

func (r *WorkflowRepository) ListTasksWithCount(ctx context.Context, scope []uuid.UUID, arg workflowdb.ListWorkflowTasksParams) ([]workflowdb.ListWorkflowTasksRow, int64, error) {
	var rows []workflowdb.ListWorkflowTasksRow
	var total int64
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListWorkflowTasks(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountWorkflowTasks(ctx, workflowdb.CountWorkflowTasksParams{
			InstanceID:  arg.InstanceID,
			AssigneeID:  arg.AssigneeID,
			PendingOnly: arg.PendingOnly,
		})
		return e
	})
	return rows, total, err
}

type TaskDecisionResult struct {
	TaskID          uuid.UUID `json:"taskId"`
	InstanceID      uuid.UUID `json:"instanceId"`
	DocumentRef     string    `json:"documentRef"`
	Decision        string    `json:"decision"`
	Comment         string    `json:"comment,omitempty"`
	InstanceStatus  string    `json:"instanceStatus"`
	TaskRowVersion  int32     `json:"taskRowVersion"`
	InstanceVersion int32     `json:"instanceRowVersion"`
}

// DecideTaskAndEmit enforces SoD (approver != requester), updates the task,
// closes the parent workflow instance once all tasks are approved (or immediately on Reject),
// and emits workflow.approval.completed with optional decision comment in one tx.
func (r *WorkflowRepository) DecideTaskAndEmit(
	ctx context.Context, scope []uuid.UUID,
	taskID, actorID, correlationID uuid.UUID,
	expectedVer *int32, decision, comment string,
) (TaskDecisionResult, error) {
	var res TaskDecisionResult
	err := r.db.WithOrgScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		task, err := q.GetWorkflowTask(ctx, taskID)
		if err != nil {
			return err
		}
		inst, err := q.GetWorkflowInstance(ctx, task.InstanceID)
		if err != nil {
			return err
		}
		if expectedVer != nil && task.RowVersion != *expectedVer {
			return ErrVersionConflict
		}
		if task.Decision != nil || inst.Status != "Pending" {
			return ErrAlreadyDecided
		}
		if decision == "Approve" && (actorID == inst.CreatedBy || actorID == task.CreatedBy) {
			return ErrSoDViolation
		}
		newTaskVer, err := q.DecideWorkflowTask(ctx, workflowdb.DecideWorkflowTaskParams{
			ID:         taskID,
			RowVersion: task.RowVersion,
			Decision:   &decision,
			UpdatedBy:  &actorID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrVersionConflict
			}
			return err
		}

		instStatus := inst.Status
		newInstVer := inst.RowVersion
		shouldClose := decision == "Reject"
		if decision == "Approve" {
			pendingLeft, err := q.CountPendingTasksByInstance(ctx, inst.ID)
			if err != nil {
				return err
			}
			shouldClose = pendingLeft == 0
		}
		if shouldClose {
			instStatus = "Approved"
			if decision == "Reject" {
				instStatus = "Rejected"
			}
			newInstVer, err = q.DecideWorkflowInstance(ctx, workflowdb.DecideWorkflowInstanceParams{
				ID:         inst.ID,
				RowVersion: inst.RowVersion,
				Status:     instStatus,
				UpdatedBy:  &actorID,
			})
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrVersionConflict
				}
				return err
			}
			evMap := map[string]any{
				"taskId":      task.ID,
				"instanceId":  inst.ID,
				"documentRef": inst.DocumentRef,
				"decision":    decision,
				"status":      instStatus,
				"approverId":  actorID,
			}
			if comment != "" {
				evMap["comment"] = comment
			}
			payload, _ := json.Marshal(evMap)
			if err := q.InsertOutbox(ctx, workflowdb.InsertOutboxParams{
				AggregateType:  "WorkflowInstance",
				AggregateID:    inst.ID,
				EventType:      "workflow.approval.completed",
				Payload:        payload,
				OrganizationID: inst.OrganizationID,
				CorrelationID:  correlationID,
			}); err != nil {
				return err
			}
		}
		res = TaskDecisionResult{
			TaskID:          task.ID,
			InstanceID:      inst.ID,
			DocumentRef:     inst.DocumentRef,
			Decision:        decision,
			Comment:         comment,
			InstanceStatus:  instStatus,
			TaskRowVersion:  newTaskVer,
			InstanceVersion: newInstVer,
		}
		return nil
	})
	return res, err
}

func (r *WorkflowRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := r.q.WithTx(tx)
	rows, err := q.DrainOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	published := 0
	var pubErr error
	for _, row := range rows {
		env := eventbus.Envelope{
			EventID:        row.ID.String(),
			EventType:      row.EventType,
			AggregateType:  row.AggregateType,
			AggregateID:    row.AggregateID.String(),
			OccurredAt:     row.CreatedAt.Time,
			CorrelationID:  row.CorrelationID.String(),
			OrganizationID: row.OrganizationID.String(),
			SchemaVersion:  "v1",
			Data:           json.RawMessage(row.Payload),
		}
		if err := pub.Publish(ctx, env); err != nil {
			log.Printf("workflow relay publish %s: %v", row.ID, err)
			pubErr = err
			break
		}
		if err := q.MarkOutboxPublished(ctx, workflowdb.MarkOutboxPublishedParams{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
		}); err != nil {
			pubErr = err
			break
		}
		published++
	}
	if published > 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}
	}
	return published, pubErr
}
