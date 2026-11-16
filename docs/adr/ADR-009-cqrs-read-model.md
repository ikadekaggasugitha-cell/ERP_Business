# ADR-009: CQRS Read Model untuk Reporting & Dashboard

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead Data Engineer  
**Terkait**: ARC-007, NFR-003 (Dashboard Rendering Time)

---

## Konteks
Dashboard eksekutif dan laporan ringkasan menuntut waktu muat instan (< 3 detik) serta agregasi data lintas modul (penjualan, piutang, buku besar, persediaan). Jika query agregasi ini dijalankan langsung ke tabel-tabel transaksi OLTP, performa input transaksi harian akan terganggu parah.

## Keputusan
Menerapkan pola **Command Query Responsibility Segregation (CQRS)**:
1. `reporting-service` (SVC-019) mengonsumsi event domain dari Kafka (misalnya `erp.gl.journal.posted.v1`, `erp.sales.order.confirmed.v1`).
2. Layanan ini membangun *read model* terdenormalisasi (seperti `rpt.gl_balance_snapshot`, `rpt.sales_summary`) di `reporting_db`.
3. Seluruh dashboard dan query pelaporan membaca dari model baca terdenormalisasi ini (atau read replica-nya), tanpa pernah menyentuh basis data transaksional.

## Rasional
- Menjamin kecepatan render dashboard tanpa mengorbankan performa penulisan transaksi di modul operasional.

## Konsekuensi
- Terdapat jeda waktu *eventual consistency* yang sangat singkat (sub-detik) antara posting transaksi dan pembaruan grafik dashboard.

