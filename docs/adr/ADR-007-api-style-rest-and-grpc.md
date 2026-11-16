# ADR-007: API Style: REST (Eksternal) + gRPC (Internal)

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead Backend Engineer  
**Terkait**: ARC-009, ARC-010, INT-013, NFR-014

---

## Konteks
Sistem membutuhkan kontrak API yang mudah dikonsumsi oleh aplikasi klien web, mobile, dan sistem pihak ketiga, namun tetap memiliki latensi minimal dan efisiensi serialisasi tinggi untuk komunikasi sinkron antar-layanan di dalam klaster privat.

## Keputusan
1. **REST + JSON (OpenAPI 3.1)**: Digunakan untuk seluruh komunikasi eksternal, klien web SPA/mobile, dan integrasi mitra melalui Kong API Gateway.
2. **gRPC (HTTP/2 + Protobuf)**: Digunakan untuk komunikasi sinkron internal antar-layanan Go di dalam klaster privat bila diperlukan query instan berlatensi rendah.
3. **Kafka Event Streaming**: Menjadi pilihan utama untuk interaksi antar-layanan yang bersifat asinkron atau memicu proses bisnis lanjutan.

## Rasional
- Menyeimbangkan kemudahan integrasi publik dengan efisiensi performa internal klaster.

## Konsekuensi
- Memerlukan pemeliharaan spesifikasi OpenAPI dan file `.proto` di pipeline CI/CD.

