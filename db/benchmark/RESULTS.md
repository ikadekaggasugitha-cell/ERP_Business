# Benchmark Results — Baseline (PostgreSQL 16, single node)

Ran `db/benchmark/benchmark_at_scale.sql`: **100,000 journal entries + 200,000 journal lines**,
50,000 AP invoices, 50,000 AR invoices, 50,000 outbox rows. Fresh DBs, `ANALYZE` before queries,
`EXPLAIN (ANALYZE, BUFFERS)`. Numbers are indicative on a laptop Docker container — the point is
the **plan shape** (index usage + partition pruning), which holds regardless of hardware.

| # | Query (critical path) | Plan (verified) | Exec time |
| - | --------------------- | --------------- | --------- |
| GL-A | Drill-down `journal_entry` by `source_document_ref` | **Index Scan** on `*_source_document_ref_idx` (each partition) | 0.244 ms |
| GL-B | Trial-balance `SUM(debit/credit)` over one month on `journal_line` | **Partition pruned → `journal_line_y2026m08`** only, HashAggregate | 4.782 ms |
| AP-A | Open items by `vendor_id` (`outstanding>0`) | **Bitmap Index Scan** `*_vendor_id_idx` | 2.004 ms |
| AP-B | Invoices due within a month | **Partition pruned → `ap_invoice_y2026m08`** only | 0.531 ms |
| AR-A | Aging by `customer_id` + `status='Overdue'` | **Bitmap Index Scan** `*_customer_id_idx` | 1.703 ms |
| OUTBOX | Relay poll `published_at IS NULL ORDER BY created_at LIMIT 100` | **Bitmap Index Scan** on partial `idx_outbox_unpublished` (other partitions pruned to 0-row scans) | 0.578 ms |

## What this proves

1. **Index scans engage at scale** — selective lookups (`source_document_ref`, `vendor_id`,
   `customer_id`, unpublished-outbox) use the intended indexes rather than sequential scans.
2. **Partition pruning works** — queries with a date/partition-key predicate (GL-B, AP-B) touch
   exactly **one** monthly partition; the planner discards the other 12.
3. **Partial index** `idx_outbox_unpublished` serves the relay poll (only ~0.2% of rows are
   pending), so the transactional-outbox drain stays cheap even as the table grows.

## Key operational guidance (confirmed by plans)

- **Always include the partition key** (`posting_date` / `invoice_date` / `movement_date` /
  `created_at`) in reporting queries to get single-partition pruning. Queries without it
  (e.g. GL-A by `source_document_ref`, outbox poll) fan out across all partitions — still cheap
  here because each fan-out leg is an index scan, but pruning is strictly better.
- Trial-balance/period-close aggregations scan one partition sequentially — expected and optimal;
  add a covering index only if a narrower projection is needed.
- Vendor/customer aging benefits from the FK indexes; consider a composite
  `(customer_id, status)` / `(vendor_id, status)` if aging reports dominate the workload.

## How to run

```bash
# fresh gl_db/ap_db/ar_db migrated, then:
psql -U postgres -f db/benchmark/benchmark_at_scale.sql          # ~100k+ rows, at-scale
psql -U postgres -f db/benchmark/explain_critical_queries.sql    # quick smoke (~few k rows)
```
