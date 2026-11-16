# ADR-002: Hybrid Multi-Tenancy & Row-Level Security

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead Security Engineer, Database Lead  
**Terkait**: ARC-004, NFR-012, NFR-020, Organization Tree First

---

## Konteks
Platform melayani banyak entitas bisnis (Holding, Sub Holding, Company, Branch, Project Site) dengan kebutuhan isolasi yang berbeda. Sebagian entitas anak perusahaan (misalnya lembaga keuangan di bawah OJK) memerlukan segregasi fisik data yang kuat, sedangkan unit operasional reguler membutuhkan efisiensi sumber daya infrastruktur.

## Keputusan
Menerapkan arsitektur **Hybrid Multi-Tenancy**:
1. **Schema-per-tenant**: Diterapkan khusus untuk tenant atau holding dengan regulasi kepatuhan ketat, kebutuhan isolasi penuh, dan SLA backup/restore independen.
2. **Shared-schema + `organization_id` + Row-Level Security (RLS)**: Diterapkan untuk mayoritas tenant standar guna efisiensi komputasi dan memori.
Penetapan mode tenancy ditentukan melalui konfigurasi saat registrasi tenant tanpa mengubah kode sumber.

## Rasional
- Menyeimbangkan isolasi keamanan data tinggi dengan efisiensi biaya infrastruktur.
- Fitur Row-Level Security di level PostgreSQL bertindak sebagai *defense-in-depth* terakhir jika terjadi kesalahan logika aplikasi.

## Konsekuensi
- Memerlukan mekanisme migrasi skema dan routing koneksi basis data yang mendukung dua pola tenancy.

