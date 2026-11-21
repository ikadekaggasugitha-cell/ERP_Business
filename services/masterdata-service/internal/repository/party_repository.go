// Package repository owns masterdata_db transactions (parties, addresses,
// items, UOMs, outbox).
package repository

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/masterdata-service/internal/masterdatadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type PartyRepository struct {
	db *platform.DB
	q  *masterdatadb.Queries
}

func NewPartyRepository(db *platform.DB) *PartyRepository {
	return &PartyRepository{db: db, q: masterdatadb.New(db.Pool)}
}

func (r *PartyRepository) CreatePartyAndEmit(ctx context.Context, scope []uuid.UUID, arg masterdatadb.CreatePartyParams, addrs []masterdatadb.CreatePartyAddressParams, ev masterdatadb.InsertOutboxParams) (masterdatadb.CreatePartyRow, error) {
	var created masterdatadb.CreatePartyRow
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateParty(ctx, arg); e != nil {
			return e
		}
		for _, a := range addrs {
			a.PartyID = created.ID
			a.OrganizationID = arg.OrganizationID
			a.CreatedBy = arg.CreatedBy
			if _, e = q.CreatePartyAddress(ctx, a); e != nil {
				return e
			}
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

func (r *PartyRepository) GetPartyWithAddresses(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (masterdatadb.MdParty, []masterdatadb.ListPartyAddressesRow, error) {
	var p masterdatadb.MdParty
	var addrs []masterdatadb.ListPartyAddressesRow
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if p, e = q.GetParty(ctx, id); e != nil {
			return e
		}
		addrs, e = q.ListPartyAddresses(ctx, id)
		return e
	})
	return p, addrs, err
}

func (r *PartyRepository) ListPartiesWithCount(ctx context.Context, scope []uuid.UUID, arg masterdatadb.ListPartiesParams) ([]masterdatadb.ListPartiesRow, int64, error) {
	var rows []masterdatadb.ListPartiesRow
	var total int64
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListParties(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountParties(ctx, arg.PartyType)
		return e
	})
	return rows, total, err
}

func (r *PartyRepository) UpdatePartyAndEmit(ctx context.Context, scope []uuid.UUID, arg masterdatadb.UpdatePartyParams, events ...masterdatadb.InsertOutboxParams) (int32, error) {
	var ver int32
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if ver, e = q.UpdateParty(ctx, arg); e != nil {
			return e
		}
		for _, ev := range events {
			if e = q.InsertOutbox(ctx, ev); e != nil {
				return e
			}
		}
		return nil
	})
	return ver, err
}

func (r *PartyRepository) DeleteParty(ctx context.Context, scope []uuid.UUID, arg masterdatadb.DeletePartyParams) (int64, error) {
	var n int64
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteParty(ctx, arg)
		return e
	})
	return n, err
}

func (r *PartyRepository) CreateItemAndEmit(ctx context.Context, scope []uuid.UUID, arg masterdatadb.CreateItemParams, ev masterdatadb.InsertOutboxParams) (masterdatadb.CreateItemRow, error) {
	var created masterdatadb.CreateItemRow
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if created, e = q.CreateItem(ctx, arg); e != nil {
			return e
		}
		one, _ := platform.ParseNumeric("1")
		if e = q.CreateItemUOM(ctx, masterdatadb.CreateItemUOMParams{
			OrganizationID:   arg.OrganizationID,
			ItemID:           created.ID,
			UomCode:          arg.BaseUom,
			ConversionToBase: one,
			IsBase:           true,
			CreatedBy:        arg.CreatedBy,
		}); e != nil {
			return e
		}
		ev.AggregateID = created.ID
		return q.InsertOutbox(ctx, ev)
	})
	return created, err
}

func (r *PartyRepository) GetItemWithUOMs(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (masterdatadb.MdItem, []masterdatadb.ListItemUOMsRow, error) {
	var item masterdatadb.MdItem
	var uoms []masterdatadb.ListItemUOMsRow
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if item, e = q.GetItem(ctx, id); e != nil {
			return e
		}
		uoms, e = q.ListItemUOMs(ctx, id)
		return e
	})
	return item, uoms, err
}

func (r *PartyRepository) ListItemsWithCount(ctx context.Context, scope []uuid.UUID, arg masterdatadb.ListItemsParams) ([]masterdatadb.ListItemsRow, int64, error) {
	var rows []masterdatadb.ListItemsRow
	var total int64
	err := r.db.WithOrgReadScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if rows, e = q.ListItems(ctx, arg); e != nil {
			return e
		}
		total, e = q.CountItems(ctx, arg.ItemType)
		return e
	})
	return rows, total, err
}

func (r *PartyRepository) UpdateItemAndEmit(ctx context.Context, scope []uuid.UUID, arg masterdatadb.UpdateItemParams, ev masterdatadb.InsertOutboxParams) (int32, error) {
	var ver int32
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if ver, e = q.UpdateItem(ctx, arg); e != nil {
			return e
		}
		return q.InsertOutbox(ctx, ev)
	})
	return ver, err
}

func (r *PartyRepository) DeleteItem(ctx context.Context, scope []uuid.UUID, arg masterdatadb.DeleteItemParams) (int64, error) {
	var n int64
	err := r.db.WithTenantWriteScope(ctx, scope, func(tx pgx.Tx) error {
		var e error
		n, e = r.q.WithTx(tx).DeleteItem(ctx, arg)
		return e
	})
	return n, err
}

func (r *PartyRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
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
			log.Printf("masterdata relay publish %s: %v", row.ID, err)
			pubErr = err
			break
		}
		if err := q.MarkOutboxPublished(ctx, masterdatadb.MarkOutboxPublishedParams{
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
