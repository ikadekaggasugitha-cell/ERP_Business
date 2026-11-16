# ADR-011: Decoupled Single-Page Application (SPA) untuk Enterprise Backoffice

**Status**: Accepted  
**Tanggal**: 2026-11-11  
**Pengambil Keputusan**: Solution Architect, Lead Frontend Engineer, Lead Platform Engineer  
**Terkait**: ADR-003 (Kong API Gateway), ADR-006 (Standard Technology Stack), ARC-007 (Kinerja Responsif), NFR-001 (API Response Time), NFR-003 (Dashboard Rendering Time), NFR-022 s.d. NFR-027 (Aksesibilitas & Lokalisasi)

---

## Konteks

Platform ERP melayani puluhan ribu pengguna internal (staf akuntansi, procurement, operasional gudang, kasir/cabang, manajemen) yang bekerja pada antarmuka transaksional kaya (*data-heavy administrative interfaces*). Antarmuka ini memerlukan:
1. Interaktivitas sangat tinggi (grid tabel ribuan baris, manipulasi jurnal bertingkat, form bertab-tab, filter kompleks yang persisten lintas navigasi).
2. Akses tertutup (*auth-gated*, zero public SEO requirements pada area backoffice).
3. Latensi navigasi nyaris nol (tidak reload HTML server per navigasi rute).
4. Pembaruan data *real-time* (status approval, notifikasi stok, progress transaksi) tanpa membebani server dengan polling berulang.

Sebelumnya, usulan awal sempat mempertimbangkan pendekatan SSR monolitik secara umum tanpa memisahkan kebutuhan antara halaman publik/marketing dan dashboard backoffice yang berat. Penggunaan SSR murni pada layar operasional kompleks menambah overhead komputasi Node.js server, meningkatkan latensi First Input Delay (FID), serta mempersulit caching statis pada edge CDN.

---

## Keputusan

Kami menetapkan arsitektur **Decoupled Single-Page Application (SPA)** sebagai standar arsitektur antarmuka operasional backoffice ERP:

1. **Pemisahan Stack**:
   - **Backoffice Transactional App**: Single-Page Application (SPA) berbasis React 19 + Vite yang di-*bundle* secara statis dan disajikan via CDN Edge (Cloudflare Pages / Nginx).
   - **Public Portal / Landing** (bila ada): Antarmuka teroptimasi statis khusus halaman publik yang membutuhkan SEO.

2. **Lapisan Manajemen State Terstruktur**:
   - **Server State (TanStack Query v5)**: Menangani fetching, background revalidation (`staleTime: 5m`), query deduplication, optimistic updates saat mutasi transaksi, dan garbage collection otomatis.
   - **Client UI State (Zustand)**: State lokal murni seperti buka/tutup sidebar, active drawer, status modal, tema (dark/light), dan preferensi layar.
   - **Form State (React Hook Form + Zod)**: Manajemen input formulir transaksional yang cepat tanpa memicu re-render seluruh halaman, dengan validasi skema bertipe kuat (*type-safe*).
   - **URL State (TanStack Router with Search Validation)**: Sinkronisasi filter pencarian, paginasi, sorting, dan rentang tanggal langsung ke URL *search parameters* agar link dapat dibagikan persis antar staf.

3. **Rendering Grid Bervolume Tinggi**:
   - Menerapkan **Virtual Scrolling** (`@tanstack/react-virtual` / virtual table primitives) pada semua tabel operasional (jurnal GL, daftar mutasi stok, invoice) sehingga mampu me-render hingga 100.000 baris transaksi tanpa degradasi frame-rate browser.

4. **Kanal Real-Time (Server-Sent Events & WebSockets)**:
   - Integrasi koneksi **Server-Sent Events (SSE)** atau **WebSocket** yang diproteksi token OIDC melalui Kong API Gateway untuk menyiarkan pembaruan status mutasi (`erp.gl.journal.posted`, `erp.workflow.approval.completed`) langsung ke client tanpa polling agresif.

5. **Decoupled Build & Edge Delivery**:
   - Frontend dibangun menggunakan Vite (ES modules, Instant HMR) menghasilkan file statis murni (`dist/`), di-deploy ke Edge / CDN dengan kompresi Brotli/Gzip dan *immutable cache headers* untuk file ter-hash.

---

## Rasional

- **Performa Transaksional Optimal**: Tidak ada latensi render server per perpindahan layar modul. Setelah asset awal ter-cache di browser, navigasi antar modul ERP terjadi seketika (*instantaneous client-side routing*).
- **Efisiensi Sumber Daya Server**: Server backend murni melayani REST JSON / SSE via Go microservices tanpa beban rendering HTML Node.js di server.
- **Kemandirian Deployment**: Tim frontend dapat melakukan rilis UI, perbaikan UX, dan pembaruan styling secara independen tanpa menyentuh deployment pod backend.
- **Resiliensi Jaringan**: Client state dan query caching memungkinkan aplikasi tetap responsif pada koneksi cabang yang berfluktuasi (*offline-first optimistic UI*).

---

## Konsekuensi

### Positif (+)
- Pengalaman pengguna sekelas aplikasi desktop (desktop-grade responsiveness) untuk staf backoffice.
- Penghematan drastis pada biaya komputasi server rendering.
- State filter dan pencarian transaksi dapat di-bookmark dan dibagikan secara akurat via URL.
- Skalabilitas CDN global: asset statis dilayani dari edge terdekat dengan cabang di berbagai negara.

### Negatif (−)
- Ukuran *initial bundle* harus dijaga ketat menggunakan dynamic imports (`React.lazy` / dynamic components) dan code splitting vendor (`manualChunks`).
- Tidak cocok untuk halaman yang memerlukan SEO publik (dimitigasi dengan memisahkan portal publik bila diperlukan).
- Membutuhkan penanganan token autentikasi (OIDC PKCE + silent refresh) yang aman di sisi browser tanpa menyimpan secret di local storage.

