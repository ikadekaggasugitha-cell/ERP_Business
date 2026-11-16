# ERP Enterprise Companion — Chrome Extension (Manifest V3)

> Ekstensi pendamping resmi ERP Enterprise untuk persetujuan dokumen instan di Side Panel, otomasi pengisian e-Faktur DJP Online, dan ekstraksi mutasi rekening korporasi untuk rekonsiliasi treasury.

---

## 1. Overview & Arsitektur

Ekstensi ini dibangun mengikuti **ADR-012 (Enterprise Chrome Extension Companion)** dengan arsitektur Manifest V3 yang aman dan *ephemeral*:

```text
┌──────────────────────────────────────────────────────────┐
│                  Chrome Extension (MV3)                  │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │        Background Service Worker (ephemeral)       │  │
│  │  • chrome.alarms sync (setiap 5 menit)             │  │
│  │  • Badge counter & chrome.storage.session/local    │  │
│  │  • Pengiriman keputusan ke Kong Gateway            │  │
│  └───────────────┬──────────────────────┬─────────────┘  │
│                  │                      │                │
│  ┌───────────────▼──────┐  ┌────────────▼─────────────┐  │
│  │   Side Panel (UI)    │  │     Content Scripts      │  │
│  │  • Tab Persetujuan   │  │ • tax-helper.js          │  │
│  │  • Tab Pajak & Bank  │  │   (https://*.pajak.go.id)│  │
│  │  • Tab Sesi & Status │  │ • bank-extractor.js      │  │
│  └───────────────┬──────┘  │   (BCA, BNI, Mandiri)    │  │
│                  │         └──────────────────────────┘  │
│  ┌───────────────▼──────┐                                │
│  │  Sandbox Iframe      │                                │
│  │  sandbox.html/.js    │  (CSP ketat, isolasi HTML)     │
│  └──────────────────────┘                                │
└──────────────────────────┬───────────────────────────────┘
                           │ HTTPS + OIDC Bearer + X-Organization-Id
                           ▼
             Kong API Gateway (/api/v1)
       (workflow-service, tax-service, treasury-service)
```

---

## 2. Struktur Direktori

```text
apps/chrome-companion/
├── manifest.json              # Deklarasi Manifest V3 (minimum Chrome 116)
├── service-worker.js          # Background Service Worker (alarms, storage, message bus)
├── icons/
│   ├── icon-16.png            # 16×16 Toolbar icon
│   ├── icon-48.png            # 48×48 Extension management icon
│   └── icon-128.png           # 128×128 Store icon
├── sidepanel/
│   ├── sidepanel.html         # Antarmuka Side Panel 3 tab
│   ├── sidepanel.css          # Stylesheet bersih tanpa ketergantungan eksternal
│   └── sidepanel.js           # Logika Side Panel patuh CSP (event delegation, no eval/inline)
├── content/
│   ├── tax-helper.js          # Deteksi & auto-fill formulir e-Faktur DJP Online
│   └── bank-extractor.js      # Ekstraksi tabel mutasi bank (BCA, BNI, Mandiri)
├── sandbox/
│   ├── sandbox.html           # Halaman terisolasi (sandbox CSP) untuk parsing HTML eksternal
│   ├── sandbox.css            # Stylesheet eksternal sandbox
│   └── sandbox.js             # Parser DOM tabel transaksi via window.postMessage
├── README.md                  # Dokumentasi ini
└── CHROMEWEBSTORE.md          # Dokumentasi publikasi & justifikasi izin CWS
```

---

## 3. Fitur Utama

1. **Quick Approvals (Persetujuan Dokumen Instan)**:
   - Menampilkan antrean PO, Journal Entry, dan Payment Voucher di Side Panel.
   - Mendukung aksi **Setujui** maupun **Tolak (dengan alasan audit)** secara langsung dengan metadata `origin: 'chrome-extension'`, header `X-Correlation-Id`, `X-Organization-Id` (RLS), dan `Idempotency-Key`.
   - Badge toolbar disinkronkan periodik setiap 5 menit melalui `chrome.alarms`.
2. **Asisten e-Faktur DJP (`pajak.go.id`)**:
   - Menyuntikkan tombol cepat mengambang (*floating badge*) di portal DJP Online.
   - Mendukung pemindaian kolom formulir (`SCAN_FIELDS`) dan injeksi terbatch via `requestAnimationFrame` (`INJECT_TAX_INVOICE`) untuk NPWP, Nama, DPP, PPN, Tanggal, dan Referensi Faktur.
3. **Treasury Bank Statement Extractor (`klikbca`, `bni`, `bankmandiri`)**:
   - Mendeteksi tabel mutasi rekening secara otomatis, menormalisasi format angka dan tanggal Indonesia (`DD/MM/YYYY`, `1.234.567,89`), lalu meneruskannya ke `treasury-cashbank-service`.
   - Menyediakan **Sandbox HTML Parser** di Side Panel untuk memparsing potongan HTML eksternal di dalam `<iframe>` ber-sandbox ketat.

---

## 4. Cara Instalasi Lokal (Developer Mode)

1. Buka Google Chrome (versi 116+) dan kunjungi `chrome://extensions/`.
2. Aktifkan toggle **Developer mode** di pojok kanan atas.
3. Klik **Load unpacked** dan pilih folder `apps/chrome-companion/`.
4. Klik ikon **ERP Enterprise Companion** pada toolbar untuk membuka **Side Panel**.
5. Buka tab **Sesi & Status** untuk menyesuaikan URL Kong API Gateway (`http://localhost:8000/api/v1`) dan *Organization Scope ID* (`11111111-1111-1111-1111-111111111111`).

---

## 5. Referensi

- [ADR-012: Enterprise Chrome Extension Companion](../../docs/adr/ADR-012-enterprise-chrome-extension-companion.md)
- [CHROMEWEBSTORE.md](./CHROMEWEBSTORE.md)
