# ADR-008: Transactional Outbox dengan Change Data Capture (CDC)

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead Data Architect  
**Terkait**: ARC-001, ARC-006, NFR-008, NFR-009

---

## Konteks
Penerbitan event ke Kafka setelah perubahan basis data rentan terhadap masalah *dual-write*: jika basis data berhasil commit namun pengiriman Kafka gagal (atau sebaliknya), data bisnis dan event akan mengalami inkonsistensi fatal.

## Keputusan
Menerapkan pola **Transactional Outbox**:
1. Setiap perubahan data domain dan pencatatan baris event ke tabel `<schema>.outbox` ditulis dalam **SATU transaksi atomik PostgreSQL**.
2. Komponen relay / CDC (misalnya Debezium atau worker outbox terisolasi) membaca baris outbox yang belum terkirim dan mempublikasikannya ke Kafka.
3. Setelah event terkonfirmasi masuk ke Kafka, baris outbox ditandai sebagai `published_at` dan dibersihkan secara berkala.

## Rasional
- Menjamin tidak ada event yang hilang (*zero event loss*) dan tidak terjadi inkonsistensi status antara database dan message broker.

## Konsekuensi
- Menambah tabel `outbox` berpartisi bulanan di setiap layanan produsen event serta proses relay publisher.

