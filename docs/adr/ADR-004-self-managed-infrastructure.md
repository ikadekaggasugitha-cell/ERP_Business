# ADR-004: Infrastruktur Self-Managed pada Kubernetes

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Chief Technology Officer, Security & Compliance Lead  
**Terkait**: ARC-003, CONS-005, ASM-013, ASM-014, NFR-004, NFR-005, NFR-028

---

## Konteks
Kepatuhan regulasi kedaulatan data (UU PDP No. 27/2022, regulasi perbankan/keuangan OJK, PSAK) dan kebutuhan kendali penuh atas tuning jaringan dan isolasi data menuntut sistem dapat berjalan di infrastruktur privat organisasi tanpa ketergantungan pada *vendor-locked managed services*.

## Keputusan
Menetapkan bahwa seluruh komponen stateful (PostgreSQL, Apache Kafka KRaft, Kong Gateway, Apicurio) dan stateless services dijalankan secara **self-managed** di atas **Kubernetes** menggunakan Operator Kubernetes (misalnya Strimzi untuk Kafka, CloudNativePG/Zalando untuk PostgreSQL), didukung oleh penyimpanan persisten berkinerja tinggi (*NVMe SSD Persistent Volumes*).

## Rasional
- Menjamin kedaulatan data penuh dan portabilitas multi-cloud atau on-premise.
- Kontrol penuh atas parameter tuning kernel, koneksi database, dan kebijakan backup/PITR.

## Konsekuensi
- Tanggung jawab operasional, patching OS/DB, kapasitas storage, dan failover HA berada sepenuhnya pada Platform SRE Team.

