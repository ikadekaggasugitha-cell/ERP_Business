# ADR-010: Observability Stack Berbasis OpenTelemetry

**Status**: Accepted  
**Tanggal**: 2026-08-05  
**Pengambil Keputusan**: Solution Architect, Lead SRE / Platform Engineer  
**Terkait**: ARC-003, NFR-004, NFR-018 (Deteksi & Respons Insiden)

---

## Konteks
Pada arsitektur microservices terdistribusi dengan 19+ layanan dan event streaming asinkron, pelacakan siklus hidup permintaan dan investigasi *root-cause* kegagalan tidak mungkin dilakukan hanya dengan log lokal per pod.

## Keputusan
Menetapkan arsitektur observabilitas terpadu berbasis **OpenTelemetry (OTel)**:
1. **Tracing Terdistribusi**: Menginjeksikan header `X-Correlation-Id` dan konteks W3C trace di setiap request HTTP, gRPC, dan payload event Kafka.
2. **Metrics**: Menggunakan Prometheus client pada seluruh layanan untuk mengekspos metrik latensi (P50, P95, P99), throughput transaksi, error rate, dan lag consumer Kafka, divisualisasikan pada Grafana.
3. **Structured Logging**: Log terstruktur berformat JSON dengan atribut wajib: `timestamp`, `level`, `service`, `correlation_id`, `organization_id`, dan pesan. Log dialirkan via Kafka / collector terpusat.
4. **Proteksi PII**: Data pribadi dan kredensial dilarang dicatat dalam plaintext di log.

## Rasional
- Standar OpenTelemetry bersifat vendor-neutral dan didukung luas oleh ekosistem cloud-native.
- Memungkinkan investigasi insiden dan pelacakan transaksi ujung-ke-ujung (*end-to-end*) dalam hitungan menit.

## Konsekuensi
- Memerlukan middleware telemetri standar di setiap layanan dan alokasi penyimpanan untuk data trace dan metrik.

