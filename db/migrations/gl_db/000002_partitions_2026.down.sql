-- gl_db — rollback 000002_partitions_2026
-- Incoming composite FK (gl.journal_line -> gl.journal_entry) creates a
-- per-partition dependency, so each partition must be DETACHed before DROP.
-- (Assumes partitions are empty of referenced rows, as in a baseline rollback.)
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2027m01; DROP TABLE IF EXISTS gl.journal_entry_y2027m01;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m12; DROP TABLE IF EXISTS gl.journal_entry_y2026m12;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m11; DROP TABLE IF EXISTS gl.journal_entry_y2026m11;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m10; DROP TABLE IF EXISTS gl.journal_entry_y2026m10;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m09; DROP TABLE IF EXISTS gl.journal_entry_y2026m09;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m08; DROP TABLE IF EXISTS gl.journal_entry_y2026m08;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m07; DROP TABLE IF EXISTS gl.journal_entry_y2026m07;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m06; DROP TABLE IF EXISTS gl.journal_entry_y2026m06;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m05; DROP TABLE IF EXISTS gl.journal_entry_y2026m05;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m04; DROP TABLE IF EXISTS gl.journal_entry_y2026m04;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m03; DROP TABLE IF EXISTS gl.journal_entry_y2026m03;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m02; DROP TABLE IF EXISTS gl.journal_entry_y2026m02;
ALTER TABLE gl.journal_entry DETACH PARTITION gl.journal_entry_y2026m01; DROP TABLE IF EXISTS gl.journal_entry_y2026m01;
