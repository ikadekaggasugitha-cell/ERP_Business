# ADR-001: Domain-Driven Microservices + Database-per-Service

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead Backend Engineer  
**Terkait**: ARC-002, ARC-005, NFR-010, NFR-012, NFR-013, BO-006

---

## Konteks
Sistem ERP mencakup puluhan modul fungsional (GL, AP, AR, Tax, Inventory, Procurement, Sales, HR, Payroll, dll.) dengan siklus rilis, beban kerja, dan kebutuhan skala yang sangat berbeda. NFR-013 menuntut *independent deployment*, sedangkan NFR-010 dan NFR-012 menuntut penskalaan horizontal dan isolasi data selektif.

## Keputusan
Sistem didekomposisi menjadi kumpulan *microservices* mandiri berdasarkan **bounded context** (Domain-Driven Design). Setiap layanan memiliki basis data PostgreSQL tersendiri (**database-per-service**). Tidak diizinkan adanya akses langsung lintas basis data; seluruh pertukaran data antar-layanan wajib melalui REST/gRPC atau *event streaming* Apache Kafka.

## Rasional
- Rilis dan penskalaan dapat dilakukan per modul secara independen tanpa risiko regresi menyeluruh.
- Batas domain bisnis menjadi tegas dan bersih.
- Kegagalan pada satu modul terisolasi dan tidak merembet ke modul lain.
- Selaras dengan prinsip *Configuration over Customization*.

## Konsekuensi
- Kompleksitas sistem terdistribusi (konsistensi *eventual* pada transaksi non-finansial).
- Memerlukan tata kelola kontrak API dan observabilitas terdistribusi yang kuat.

