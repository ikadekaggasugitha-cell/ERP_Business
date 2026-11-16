# ADR-003: Kong API Gateway dan Apicurio Registry

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead Infrastructure Engineer  
**Terkait**: ARC-008, ARC-009, INT-013 s.d. INT-016, NFR-014

---

## Konteks
Dibutuhkan *single entry point* yang aman dan terpadu untuk autentikasi OIDC/OAuth2, otorisasi token JWT, routing ke microservices, pembatasan kuota (*rate limiting*), dan penanganan versi API. Di sisi asynchronous, diperlukan tata kelola skema event Kafka agar perubahan payload tidak mematahkan konsumen lama.

## Keputusan
1. Menetapkan **Kong** sebagai API Gateway resmi untuk semua traffic eksternal/klien.
2. Menetapkan **Apicurio Registry** sebagai schema registry untuk tata kelola skema event (Avro/Protobuf) dengan aturan kompatibilitas `BACKWARD`.

## Rasional
- Kong memiliki ekosistem plugin keamanan, performa tinggi (OpenResty), dan integrasi OIDC yang matang.
- Apicurio menyediakan validasi kompatibilitas skema otomatis di CI/CD pipeline untuk mencegah *breaking changes*.

## Konsekuensi
- Kong dan Apicurio menjadi simpul kritikal yang wajib berjalan dalam mode High Availability (HA multi-replica).

