// Package repository provides RLS-enforced Language, Translation and Locale
// Format persistence plus the transactional outbox relay.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/localization-service/internal/localizationdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

var systemOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

type LocalizationRepository struct {
	db *platform.DB
	q  *localizationdb.Queries
}

func NewLocalizationRepository(db *platform.DB) *LocalizationRepository {
	var q *localizationdb.Queries
	if db != nil && db.Pool != nil {
		q = localizationdb.New(db.Pool)
	}
	return &LocalizationRepository{db: db, q: q}
}

// ScopeWithSystemOrg returns a copy of scope that includes the global system
// organization (seeded languages/locale formats) when scope is non-empty.
func ScopeWithSystemOrg(scope []uuid.UUID) []uuid.UUID {
	if len(scope) == 0 {
		return nil
	}
	out := make([]uuid.UUID, len(scope), len(scope)+1)
	copy(out, scope)
	for _, id := range out {
		if id == systemOrgID {
			return out
		}
	}
	return append(out, systemOrgID)
}

// withOrgTx fails closed on an empty scope, then runs fn under RLS scope+system org.
func (r *LocalizationRepository) withOrgTx(ctx context.Context, scope []uuid.UUID, fn func(pgx.Tx) error) error {
	if len(scope) == 0 {
		return errors.New("org scope must not be empty")
	}
	return r.db.WithOrgScope(ctx, ScopeWithSystemOrg(scope), fn)
}

func emit(ctx context.Context, q *localizationdb.Queries, aggType, eventType string, id, org, corr uuid.UUID, payload map[string]any) error {
	payload["id"] = id.String()
	payload["organizationId"] = org.String()
	b, _ := json.Marshal(payload)
	return q.InsertOutbox(ctx, localizationdb.InsertOutboxParams{
		AggregateType: aggType, AggregateID: id, EventType: eventType,
		Payload: b, OrganizationID: org, CorrelationID: corr,
	})
}

func stale(what string) error {
	return httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau "+what+" tidak ditemukan")
}

// ---------------- Language ----------------

func (r *LocalizationRepository) CreateLanguage(ctx context.Context, scope []uuid.UUID, arg localizationdb.CreateLanguageParams, corr uuid.UUID) (localizationdb.CreateLanguageRow, error) {
	var out localizationdb.CreateLanguageRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.CreateLanguage(ctx, arg); e != nil {
			return e
		}
		if arg.IsFallback {
			if e = q.ClearFallbackExcept(ctx, localizationdb.ClearFallbackExceptParams{ID: out.ID, OrganizationID: arg.OrganizationID}); e != nil {
				return e
			}
		}
		return emit(ctx, q, "Language", "localization.language.updated", out.ID, arg.OrganizationID, corr,
			map[string]any{"code": arg.Code, "isFallback": arg.IsFallback, "isActive": arg.IsActive})
	})
	if err != nil {
		return out, httpx.MapDBError(err, "language with this code already exists in organization")
	}
	return out, nil
}

func (r *LocalizationRepository) GetLanguage(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (localizationdb.LocLanguage, error) {
	var out localizationdb.LocLanguage
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetLanguage(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("language not found")
		}
		return e
	})
	return out, err
}

func (r *LocalizationRepository) ListLanguages(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]localizationdb.LocLanguage, int64, error) {
	var rows []localizationdb.LocLanguage
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountLanguages(ctx); e != nil {
			return e
		}
		rows, e = q.ListLanguages(ctx, localizationdb.ListLanguagesParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *LocalizationRepository) UpdateLanguage(ctx context.Context, scope []uuid.UUID, arg localizationdb.UpdateLanguageParams, corr uuid.UUID) (int32, error) {
	var ver int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		ver, e = q.UpdateLanguage(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return stale("language")
		}
		if e != nil {
			return e
		}
		if arg.IsFallback {
			if e = q.ClearFallbackExcept(ctx, localizationdb.ClearFallbackExceptParams{ID: arg.ID, OrganizationID: scope[0]}); e != nil {
				return e
			}
		}
		return emit(ctx, q, "Language", "localization.language.updated", arg.ID, scope[0], corr,
			map[string]any{"isFallback": arg.IsFallback, "isActive": arg.IsActive})
	})
	if err != nil {
		return 0, httpx.MapDBError(err, "")
	}
	return ver, nil
}

// ---------------- Translation ----------------

func (r *LocalizationRepository) CreateTranslation(ctx context.Context, scope []uuid.UUID, arg localizationdb.CreateTranslationParams, corr uuid.UUID) (localizationdb.CreateTranslationRow, error) {
	var out localizationdb.CreateTranslationRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.CreateTranslation(ctx, arg); e != nil {
			return e
		}
		return emit(ctx, q, "Translation", "localization.bundle.updated", out.ID, arg.OrganizationID, corr,
			map[string]any{"languageCode": arg.LanguageCode, "resourceKey": arg.ResourceKey})
	})
	if err != nil {
		return out, httpx.MapDBError(err, "translation key already exists for this language")
	}
	return out, nil
}

func (r *LocalizationRepository) GetTranslation(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (localizationdb.LocTranslation, error) {
	var out localizationdb.LocTranslation
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetTranslation(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("translation not found")
		}
		return e
	})
	return out, err
}

func (r *LocalizationRepository) ListTranslations(ctx context.Context, scope []uuid.UUID, lang, module *string, limit, offset int32) ([]localizationdb.LocTranslation, int64, error) {
	var rows []localizationdb.LocTranslation
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountTranslations(ctx, localizationdb.CountTranslationsParams{LanguageCode: lang, Module: module}); e != nil {
			return e
		}
		rows, e = q.ListTranslations(ctx, localizationdb.ListTranslationsParams{LanguageCode: lang, Module: module, Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *LocalizationRepository) UpdateTranslation(ctx context.Context, scope []uuid.UUID, arg localizationdb.UpdateTranslationParams, corr uuid.UUID) (int32, error) {
	var ver int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		ver, e = q.UpdateTranslation(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return stale("translation")
		}
		if e != nil {
			return e
		}
		return emit(ctx, q, "Translation", "localization.bundle.updated", arg.ID, scope[0], corr, map[string]any{})
	})
	if err != nil {
		return 0, httpx.MapDBError(err, "")
	}
	return ver, nil
}

func (r *LocalizationRepository) DeleteTranslation(ctx context.Context, scope []uuid.UUID, arg localizationdb.DeleteTranslationParams, corr uuid.UUID) error {
	return r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		n, e := q.DeleteTranslation(ctx, arg)
		if e != nil {
			return e
		}
		if n == 0 {
			return stale("translation")
		}
		return emit(ctx, q, "Translation", "localization.bundle.updated", arg.ID, scope[0], corr, map[string]any{"deleted": true})
	})
}

// GetBundle returns key→value for languageCode, overlaid on the fallback
// language bundle (org entries override global; requested language overrides fallback).
func (r *LocalizationRepository) GetBundle(ctx context.Context, scope []uuid.UUID, languageCode string) (map[string]string, error) {
	out := map[string]string{}
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, e := q.GetLanguageByCode(ctx, languageCode); errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("language not found")
		} else if e != nil {
			return e
		}
		if fb, e := q.GetFallbackLanguage(ctx); e == nil && fb.Code != languageCode {
			rows, e := q.ListTranslationsByLanguage(ctx, fb.Code)
			if e != nil {
				return e
			}
			for _, row := range rows {
				out[row.ResourceKey] = row.ResourceValue
			}
		} else if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		rows, e := q.ListTranslationsByLanguage(ctx, languageCode)
		if e != nil {
			return e
		}
		for _, row := range rows {
			out[row.ResourceKey] = row.ResourceValue
		}
		return nil
	})
	return out, err
}

// ---------------- Locale Format ----------------

func (r *LocalizationRepository) CreateLocaleFormat(ctx context.Context, scope []uuid.UUID, arg localizationdb.CreateLocaleFormatParams, corr uuid.UUID) (localizationdb.CreateLocaleFormatRow, error) {
	var out localizationdb.CreateLocaleFormatRow
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if out, e = q.CreateLocaleFormat(ctx, arg); e != nil {
			return e
		}
		return emit(ctx, q, "LocaleFormat", "localization.locale_format.updated", out.ID, arg.OrganizationID, corr,
			map[string]any{"localeCode": arg.LocaleCode})
	})
	if err != nil {
		return out, httpx.MapDBError(err, "locale format already exists in organization")
	}
	return out, nil
}

func (r *LocalizationRepository) GetLocaleFormat(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (localizationdb.LocLocaleFormat, error) {
	var out localizationdb.LocLocaleFormat
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		var e error
		out, e = r.q.WithTx(tx).GetLocaleFormat(ctx, id)
		if errors.Is(e, pgx.ErrNoRows) {
			return httpx.NotFound("locale format not found")
		}
		return e
	})
	return out, err
}

func (r *LocalizationRepository) ListLocaleFormats(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]localizationdb.LocLocaleFormat, int64, error) {
	var rows []localizationdb.LocLocaleFormat
	var total int64
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		if total, e = q.CountLocaleFormats(ctx); e != nil {
			return e
		}
		rows, e = q.ListLocaleFormats(ctx, localizationdb.ListLocaleFormatsParams{Limit: limit, Offset: offset})
		return e
	})
	return rows, total, err
}

func (r *LocalizationRepository) UpdateLocaleFormat(ctx context.Context, scope []uuid.UUID, arg localizationdb.UpdateLocaleFormatParams, corr uuid.UUID) (int32, error) {
	var ver int32
	err := r.withOrgTx(ctx, scope, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var e error
		ver, e = q.UpdateLocaleFormat(ctx, arg)
		if errors.Is(e, pgx.ErrNoRows) {
			return stale("locale format")
		}
		if e != nil {
			return e
		}
		return emit(ctx, q, "LocaleFormat", "localization.locale_format.updated", arg.ID, scope[0], corr, map[string]any{})
	})
	if err != nil {
		return 0, httpx.MapDBError(err, "")
	}
	return ver, nil
}

// DrainAndPublish drains loc.outbox under FOR UPDATE SKIP LOCKED inside a tx.
// Partial publish failures commit already-published rows (never roll back the batch).
func (r *LocalizationRepository) DrainAndPublish(ctx context.Context, limit int32, pub *eventbus.Publisher) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin outbox drain: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
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
		if e := pub.Publish(ctx, env); e != nil {
			pubErr = e
			break
		}
		if e := q.MarkOutboxPublished(ctx, localizationdb.MarkOutboxPublishedParams{ID: row.ID, CreatedAt: row.CreatedAt}); e != nil {
			pubErr = e
			break
		}
		published++
	}
	if published > 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit outbox drain: %w", err)
		}
		committed = true
	}
	return published, pubErr
}

