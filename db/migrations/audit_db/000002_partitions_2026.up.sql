-- =============================================================================
-- audit_db — Monthly RANGE partitions for audit.audit_log (DBS-013)
-- Baseline year 2026 + Jan 2027 headroom. Future partitions should be
-- provisioned automatically (pg_partman / scheduled job) — SAD BAB 6.8.
-- golang-migrate: 000002_partitions_2026.up.sql
-- =============================================================================

CREATE TABLE audit.audit_log_y2026m01 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE audit.audit_log_y2026m02 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE audit.audit_log_y2026m03 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE audit.audit_log_y2026m04 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE audit.audit_log_y2026m05 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE audit.audit_log_y2026m06 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE audit.audit_log_y2026m07 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE audit.audit_log_y2026m08 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE audit.audit_log_y2026m09 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE audit.audit_log_y2026m10 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE audit.audit_log_y2026m11 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE audit.audit_log_y2026m12 PARTITION OF audit.audit_log FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');
CREATE TABLE audit.audit_log_y2027m01 PARTITION OF audit.audit_log FOR VALUES FROM ('2027-01-01') TO ('2027-02-01');
