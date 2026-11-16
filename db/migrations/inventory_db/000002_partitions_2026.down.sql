-- inventory_db — rollback 000002_partitions_2026
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2027m01; DROP TABLE IF EXISTS inv.stock_movement_y2027m01;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m12; DROP TABLE IF EXISTS inv.stock_movement_y2026m12;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m11; DROP TABLE IF EXISTS inv.stock_movement_y2026m11;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m10; DROP TABLE IF EXISTS inv.stock_movement_y2026m10;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m09; DROP TABLE IF EXISTS inv.stock_movement_y2026m09;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m08; DROP TABLE IF EXISTS inv.stock_movement_y2026m08;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m07; DROP TABLE IF EXISTS inv.stock_movement_y2026m07;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m06; DROP TABLE IF EXISTS inv.stock_movement_y2026m06;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m05; DROP TABLE IF EXISTS inv.stock_movement_y2026m05;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m04; DROP TABLE IF EXISTS inv.stock_movement_y2026m04;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m03; DROP TABLE IF EXISTS inv.stock_movement_y2026m03;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m02; DROP TABLE IF EXISTS inv.stock_movement_y2026m02;
ALTER TABLE inv.stock_movement DETACH PARTITION inv.stock_movement_y2026m01; DROP TABLE IF EXISTS inv.stock_movement_y2026m01;
