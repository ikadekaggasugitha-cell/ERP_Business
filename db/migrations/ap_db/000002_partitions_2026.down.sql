-- ap_db — rollback 000002_partitions_2026
-- Incoming composite FKs (ap_invoice_line, ap_payment_allocation) create a
-- per-partition dependency; DETACH each partition before DROP.
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2027m01; DROP TABLE IF EXISTS ap.ap_invoice_y2027m01;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m12; DROP TABLE IF EXISTS ap.ap_invoice_y2026m12;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m11; DROP TABLE IF EXISTS ap.ap_invoice_y2026m11;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m10; DROP TABLE IF EXISTS ap.ap_invoice_y2026m10;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m09; DROP TABLE IF EXISTS ap.ap_invoice_y2026m09;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m08; DROP TABLE IF EXISTS ap.ap_invoice_y2026m08;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m07; DROP TABLE IF EXISTS ap.ap_invoice_y2026m07;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m06; DROP TABLE IF EXISTS ap.ap_invoice_y2026m06;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m05; DROP TABLE IF EXISTS ap.ap_invoice_y2026m05;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m04; DROP TABLE IF EXISTS ap.ap_invoice_y2026m04;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m03; DROP TABLE IF EXISTS ap.ap_invoice_y2026m03;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m02; DROP TABLE IF EXISTS ap.ap_invoice_y2026m02;
ALTER TABLE ap.ap_invoice DETACH PARTITION ap.ap_invoice_y2026m01; DROP TABLE IF EXISTS ap.ap_invoice_y2026m01;
