# ADR-005: Selective Exactly-Once Event Delivery

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead Financial Architect, Event Streaming Specialist  
**Terkait**: ARC-001, ARC-006, NFR-008 (Transaction Consistency)

---

## Konteks
NFR-008 menuntut konsistensi absolut pada transaksi akuntansi dan finansial. Namun, menerapkan semantik *Exactly-Once* (EOS Kafka transactions) secara menyeluruh ke seluruh sistem akan menurunkan throughput dan menambah kompleksitas konsumen pada modul non-finansial.

## Keputusan
Menerapkan pendekatan **Selective Exactly-Once Semantics**:
1. **Exactly-Once Semantics (EOS)**: Diterapkan secara ketat hanya pada proses finansial berkonsistensi absolut, yaitu **General Ledger, Journal Posting, Payment Settlement, dan Inventory Stock Reservation**.
2. **At-Least-Once + Idempotensi**: Diterapkan pada modul operasional umum, Reporting, Dashboard, Notifikasi, Audit Log, dan Master Data.

## Rasional
- Menjaga integritas saldo buku besar dan akurasi kasir/bank tanpa kompromi.
- Memaksimalkan throughput pemrosesan untuk jutaan event non-finansial harian.

## Konsekuensi
- Seluruh konsumen event wajib mengimplementasikan tabel pencatatan idempotensi (`processed_events`) sebagai lapisan pertahanan ganda.

