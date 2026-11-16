# Extended Services Roadmap: SVC-020 Through SVC-029

**Document Version**: 1.0.0  
**Status**: Approved Architecture Roadmap  
**Effective Date**: 2026-11-13  
**Target Release**: Phase 6 / Milestone M10  
**References**:
- ADR-001 (Microservices & Database-Per-Service)
- ADR-002 (Hybrid Multi-Tenancy & Row-Level Security)
- ADR-008 (Transactional Outbox & Change Data Capture)
- ADR-014 (Tri-Currency Multi-Country Real-Time Accounting Engine)
- DBDD BAB 9 (Extended Domain Schema: `docs/ERP-DB/BAB 9 Extended Domain Schema`)
- BRD BAB 9 (Extended Domain Business Rules: BR-586..825)

---

## 1. Executive Summary

This roadmap formalizes the architecture, domain boundaries, data models, API contracts, Kafka asynchronous events, and deployment specifications for the **10 Extended Microservices** (`SVC-020` through `SVC-029`).

Together with the 20 active core services (`SVC-001` through `SVC-019` plus `SVC-010-CUR`), these extended services complete the 30-service enterprise topology of the ERP platform. Each service maintains an independent PostgreSQL database with strict Row-Level Security (`current_org_scope()`), transactional outbox event publishing, and dual-period/tri-currency support.

---

## 2. Extended Microservices Catalog

| Service ID | Service Name | Internal Port | Dedicated Database | Primary Schema | Domain & Business Rules |
|---|---|---|---|---|---|
| **SVC-020** | `fixedasset-service` | 8101 | `fixedasset_db` | `fa` | Fixed asset master, capitalization, monthly depreciation runs (Straight-Line, Declining Balance, Units-of-Production), asset transfers, revaluations, impairments, and retirements. (BR-586..595) |
| **SVC-021** | `dimension-service` | 8102 | `dimension_db` | `cc`, `pc` | Cost Center (`cc`) and Profit Center (`pc`) hierarchies, allocation rules, dimension tagging across journal entries and operational documents. (BR-616..635) |
| **SVC-022** | `budget-service` | 8103 | `budget_db` | `bud` | Annual/quarterly budget creation, revisions, real-time commitment control (hard/soft budget enforcement against PR/PO/GL). (BR-636..645) |
| **SVC-023** | `financialclose-service` | 8104 | `financialclose_db` | `fcl` | Period-end closing checklist, automated foreign exchange revaluation (PSAK 10 / IAS 21), accrual reversal, intercompany elimination reconciliation, period lock orchestrator. (BR-656..665) |
| **SVC-024** | `consolidation-service` | 8105 | `consolidation_db` | `cons` | Multi-tier corporate consolidation, minority interest calculation, currency translation reserve (CTA), consolidated financial statements. (BR-666..675) |
| **SVC-025** | `project-service` | 8106 | `project_db` | `prj` | Project Work Breakdown Structure (WBS), project milestones, budgets, time & expense tracking, project billing, revenue recognition. (BR-726..735) |
| **SVC-026** | `asset-service` | 8107 | `asset_db` | `ast` | Physical asset tracking, barcode/RFID integration, equipment custody, geolocation, warranty & maintenance scheduling link. (BR-736..745) |
| **SVC-027** | `maintenance-service` | 8108 | `maintenance_db` | `mnt` | Preventive/corrective maintenance work orders, spare part reservations, machine downtime tracking, mean time between failures (MTBF). (BR-786..795) |
| **SVC-028** | `crm-service` | 8109 | `crm_db` | `crm` | Lead management, customer opportunity pipeline, quotes, customer interaction histories, sales rep territories. (BR-776..785) |
| **SVC-029** | `localization-service` | 8110 | `localization_db` | `loc` | Country-specific statutory localization rules, multi-language UI translation bundles, tax jurisdiction packs, statutory reporting templates. (BR-816..825) |

---

## 3. Service Deep-Dives: APIs, Schemas & Event Architecture

### 3.1 SVC-020: `fixedasset-service`
- **Database**: `fixedasset_db` (Schema `fa`)
- **Key Tables**: `fa.asset_category`, `fa.fixed_asset`, `fa.depreciation_schedule`, `fa.depreciation_run`, `fa.asset_transaction`, `fa.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/asset-categories`, `POST /api/v1/asset-categories`
  - `GET /api/v1/fixed-assets`, `POST /api/v1/fixed-assets`, `GET /api/v1/fixed-assets/{id}`
  - `POST /api/v1/fixed-assets/{id}/capitalize`
  - `POST /api/v1/depreciation-runs` (executes batch depreciation)
  - `POST /api/v1/fixed-assets/{id}/dispose`
  - `POST /api/v1/fixed-assets/{id}/transfer`
- **Produced Kafka Events**:
  - `erp.fa.asset.created.v1`
  - `erp.fa.asset.capitalized.v1`
  - `erp.fa.depreciation.posted.v1` (consumed by `gl-service` to post journal entry)
  - `erp.fa.asset.disposed.v1`
- **Consumed Kafka Events**:
  - `erp.ap.invoice.recorded.v1` (creates candidate asset from capital purchase line)

### 3.2 SVC-021: `dimension-service`
- **Database**: `dimension_db` (Schemas `cc`, `pc`)
- **Key Tables**: `cc.cost_center_hierarchy`, `cc.cost_center`, `cc.allocation_rule`, `pc.profit_center_hierarchy`, `pc.profit_center`, `cc.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/cost-centers`, `POST /api/v1/cost-centers`
  - `GET /api/v1/profit-centers`, `POST /api/v1/profit-centers`
  - `GET /api/v1/allocation-rules`, `POST /api/v1/allocation-rules`
  - `POST /api/v1/allocations/execute`
- **Produced Kafka Events**:
  - `erp.dimension.cost_center.upserted.v1`
  - `erp.dimension.profit_center.upserted.v1`
  - `erp.dimension.allocation.executed.v1`

### 3.3 SVC-022: `budget-service`
- **Database**: `budget_db` (Schema `bud`)
- **Key Tables**: `bud.budget_version`, `bud.budget_header`, `bud.budget_line`, `bud.budget_commitment`, `bud.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/budget-versions`, `POST /api/v1/budget-versions`
  - `GET /api/v1/budgets`, `POST /api/v1/budgets`, `PUT /api/v1/budgets/{id}`
  - `POST /api/v1/budgets/check-commitment` (high-throughput commitment validation)
  - `GET /api/v1/budgets/utilization-report`
- **Produced Kafka Events**:
  - `erp.budget.version.approved.v1`
  - `erp.budget.threshold.exceeded.v1`
- **Consumed Kafka Events**:
  - `erp.procurement.pr.created.v1` (reserves soft budget)
  - `erp.procurement.po.approved.v1` (reserves hard commitment)
  - `erp.ap.invoice.recorded.v1` (converts commitment to actual expenditure)

### 3.4 SVC-023: `financialclose-service`
- **Database**: `financialclose_db` (Schema `fcl`)
- **Key Tables**: `fcl.closing_task`, `fcl.closing_run`, `fcl.fx_revaluation_run`, `fcl.period_lock_log`, `fcl.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/closing-tasks`, `POST /api/v1/closing-tasks`
  - `POST /api/v1/closing-runs/initiate`
  - `POST /api/v1/closing-runs/{id}/fx-revaluation`
  - `POST /api/v1/closing-runs/{id}/lock-period`
- **Produced Kafka Events**:
  - `erp.fcl.period.closing_initiated.v1`
  - `erp.fcl.fx_revaluation.completed.v1`
  - `erp.fcl.period.locked.v1` (consumed by `coa-service` to transition fiscal period to Locked)

### 3.5 SVC-024: `consolidation-service`
- **Database**: `consolidation_db` (Schema `cons`)
- **Key Tables**: `cons.consolidation_group`, `cons.group_member`, `cons.elimination_rule`, `cons.consolidation_run`, `cons.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/consolidation-groups`, `POST /api/v1/consolidation-groups`
  - `GET /api/v1/elimination-rules`, `POST /api/v1/elimination-rules`
  - `POST /api/v1/consolidation-runs/execute`
  - `GET /api/v1/consolidation-reports/balance-sheet`
- **Produced Kafka Events**:
  - `erp.consolidation.run.completed.v1`
  - `erp.consolidation.elimination.discrepancy.v1`

### 3.6 SVC-025: `project-service`
- **Database**: `project_db` (Schema `prj`)
- **Key Tables**: `prj.project`, `prj.wbs_element`, `prj.project_milestone`, `prj.project_budget`, `prj.timesheet_entry`, `prj.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/projects`, `POST /api/v1/projects`, `GET /api/v1/projects/{id}`
  - `POST /api/v1/projects/{id}/wbs`, `GET /api/v1/projects/{id}/wbs`
  - `POST /api/v1/timesheets`, `GET /api/v1/timesheets`
  - `POST /api/v1/projects/{id}/billing-schedule`
- **Produced Kafka Events**:
  - `erp.project.created.v1`
  - `erp.project.milestone.achieved.v1` (triggers AR invoice generation)
  - `erp.project.cost.posted.v1`

### 3.7 SVC-026: `asset-service`
- **Database**: `asset_db` (Schema `ast`)
- **Key Tables**: `ast.equipment`, `ast.asset_tag`, `ast.custody_transfer`, `ast.location_history`, `ast.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/equipment`, `POST /api/v1/equipment`
  - `POST /api/v1/equipment/{id}/transfer-custody`
  - `GET /api/v1/equipment/{id}/history`
- **Produced Kafka Events**:
  - `erp.asset.equipment.registered.v1`
  - `erp.asset.custody.transferred.v1`

### 3.8 SVC-027: `maintenance-service`
- **Database**: `maintenance_db` (Schema `mnt`)
- **Key Tables**: `mnt.maintenance_plan`, `mnt.work_order`, `mnt.work_order_task`, `mnt.part_requirement`, `mnt.downtime_log`, `mnt.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/maintenance-plans`, `POST /api/v1/maintenance-plans`
  - `GET /api/v1/maintenance-work-orders`, `POST /api/v1/maintenance-work-orders`
  - `POST /api/v1/maintenance-work-orders/{id}/complete`
- **Produced Kafka Events**:
  - `erp.maintenance.workorder.created.v1`
  - `erp.maintenance.sparepart.reserved.v1` (consumed by `inventory-service`)
  - `erp.maintenance.workorder.completed.v1`

### 3.9 SVC-028: `crm-service`
- **Database**: `crm_db` (Schema `crm`)
- **Key Tables**: `crm.lead`, `crm.opportunity`, `crm.opportunity_stage`, `crm.quotation`, `crm.customer_activity`, `crm.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/leads`, `POST /api/v1/leads`
  - `GET /api/v1/opportunities`, `POST /api/v1/opportunities`
  - `POST /api/v1/opportunities/{id}/convert-to-order`
- **Produced Kafka Events**:
  - `erp.crm.lead.qualified.v1`
  - `erp.crm.opportunity.won.v1` (triggers `sales-service` sales order)

### 3.10 SVC-029: `localization-service`
- **Database**: `localization_db` (Schema `loc`)
- **Key Tables**: `loc.locale`, `loc.translation_bundle`, `loc.statutory_rule`, `loc.withholding_tax_rule`, `loc.outbox`.
- **Key REST Endpoints**:
  - `GET /api/v1/locales`
  - `GET /api/v1/translation-bundles/{locale}`
  - `GET /api/v1/statutory-rules/{countryCode}`
- **Produced Kafka Events**:
  - `erp.localization.bundle.updated.v1`
  - `erp.localization.statutory_rule.updated.v1`

---

## 4. Platform Cross-Cutting Services

In addition to domain services, the extended enterprise ecosystem includes two cross-cutting platform utilities:
1. **`notification-service`**:
   - Multi-channel notification dispatch (Email via SES/SMTP, Web Push via Chrome Companion / Service Worker, SSE events, and Slack/Teams webhooks).
   - Consumes workflow approval events, budget overrun alerts, and journal posting failures.
2. **`document-service`**:
   - Secure S3/MinIO/Blob object storage with presigned URLs.
   - Attaches scanned tax invoices, purchase contracts, delivery receipts, and asset photos to business transactions with cryptographical hash verification.

---

## 5. Phase 6 / Milestone M10 Implementation Sequence

1. **Sprint 1 (Infrastructure & Dimensions)** — ✅ kode selesai (live smoke Step 14 pending; allocation `execute` & statutory rules ditunda karena DDL `loc.*` aktual berbeda dari katalog §3.10):
   - Scaffold `services/dimension-service` and `services/localization-service`.
   - Run database migrations `000001_init.up.sql` on `dimension_db` and `localization_db`.
   - Wire API gateway proxy rules and Docker Compose ports `8102` and `8110`.
2. **Sprint 2 (Asset Lifecycle & Budgets)**:
   - Scaffold `services/fixedasset-service` and `services/budget-service`.
   - Implement budget commitment hook into `procurement-service` PR/PO workflows.
   - Implement monthly depreciation calculation and Kafka outbox relay to `gl-service`.
3. **Sprint 3 (Financial Close & Consolidation)**:
   - Scaffold `services/financialclose-service` and `services/consolidation-service`.
   - Implement foreign exchange closing rate revaluation (PSAK 10 / IAS 21) consuming rates from `currency-service`.
4. **Sprint 4 (Projects, Maintenance, CRM & Operations)**:
   - Scaffold `services/project-service`, `services/asset-service`, `services/maintenance-service`, and `services/crm-service`.
   - Implement project milestone billing to `ar-service` and spare part reservation to `inventory-service`.
5. **Sprint 5 (End-to-End Verification Gate)**:
   - Add ports 8101..8110 to `deploy/smoke-test.sh`.
   - Extend Cortex Framework Dataform models for Fixed Assets, Budgets, and Project Accounting.

