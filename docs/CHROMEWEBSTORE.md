# Chrome Web Store Listing & Compliance — ERP Enterprise Companion

Single source of truth for the **Enterprise Chrome Extension Companion (Manifest V3)** listing metadata, permissions justifications, privacy disclosures, and store readiness.

---

## 1. Store Listing Metadata

| Field | Value |
|---|---|
| **Extension Name** | ERP Enterprise Companion |
| **Short Description** | Asisten resmi ERP Enterprise untuk validasi e-Faktur, rekonsiliasi kas/bank, dan approval workflow instan. |
| **Category** | Productivity / Business Tools |
| **Default Locale** | id (Bahasa Indonesia) / en (English) |
| **Distribution** | Enterprise Private Listing (Google Workspace Domain Restricted) |

### Detailed Description
```text
ERP Enterprise Companion adalah ekstensi browser resmi yang menghubungkan sistem ERP Enterprise langsung ke alur kerja harian staf akuntansi, perpajakan, perbendaharaan, dan manajer cabang.

Fitur Utama:
1. In-Context Tax & E-Faktur Helper:
   - Validasi data faktur penjualan dan pembelian secara otomatis langsung di portal resmi DJP Online / e-Faktur tanpa input manual berulang.
   - Penarikan kode QR validasi dan nomor NTTE langsung tersinkronisasi ke basis data pajak ERP.

2. Corporate Banking Reconciler:
   - Ekstraksi rekening koran digital secara aman dari sesi internet banking korporasi yang sedang dibuka oleh staf treasury.
   - Pencocokan mutasi kas/bank secara otomatis (auto-reconciliation) dengan buku besar perbendaharaan.

3. Executive Side Panel Quick Approvals:
   - Akses antrean persetujuan dokumen kritis (Purchase Order, Journal Entry, Payroll Run) melalui Google Chrome Side Panel.
   - Setujui atau tolak pengajuan transaksi hanya dengan satu klik tanpa harus meninggalkan tab pekerjaan yang sedang aktif.

4. Warehouse & Logistics Hardware Web-Bridge:
   - Menghubungkan pemindai barcode USB/Bluetooth dan reader RFID gudang melalui Web Serial API ke form input ERP.

Keamanan & Privasi:
- Berjalan di atas standar Google Chrome Manifest V3 dengan background service worker yang aman dan efisien.
- Menggunakan autentikasi OAuth2 Authorization Code + PKCE terintegrasi dengan Identity Provider korporasi.
- Tidak pernah menyimpan kredensial atau data pribadi secara permanen di storage browser.
```

---

## 2. Permissions Justification (Manifest V3)

| Permission | Technical Requirement | Plain-English Justification for Review Team |
|---|---|---|
| `sidePanel` | `chrome.sidePanel` | Digunakan untuk menampilkan antarmuka persetujuan cepat (Quick Approvals) dokumen operasional ERP di sisi browser tanpa mengganggu tab kerja utama. |
| `storage` | `chrome.storage.session` & `chrome.storage.local` | Digunakan untuk menyimpan sesi aktif token OAuth PKCE dan preferensi tampilan user secara terisolasi. |
| `identity` | `chrome.identity.launchWebAuthFlow` | Diperlukan untuk alur autentikasi Single Sign-On (SSO) OAuth2/OIDC yang aman dengan server otorisasi ERP enterprise. |
| `alarms` | `chrome.alarms` | Diperlukan untuk sinkronisasi periodik jumlah antrean dokumen pending approval pada background service worker yang bersifat ephemeral. |

### Host Permissions
| Host Pattern | Justification |
|---|---|
| `https://api.erp.example/*` | Endpoint resmi API Gateway Kong untuk komunikasi REST API dan Server-Sent Events (SSE). |
| `https://*.pajak.go.id/*` | Domain portal resmi DJP Online untuk pembacaan status dan injeksi data faktur pajak elektronik. |

---

## 3. Privacy & Data Use Disclosure

- **Data Collection**:
  - *Authentication Info*: Token akses OAuth2 berumur pendek (short-lived JWT) untuk identifikasi pengguna.
  - *Financial & Business Data*: Nomor dokumen transaksi, nomor seri faktur pajak, dan nilai transaksi yang sedang diproses pengguna.
- **Data Transmission**: Seluruh data dikirimkan secara terenkripsi (TLS 1.3) langsung ke API Gateway Kong milik organisasi. Data tidak pernah dikirim ke server pihak ketiga manapun.
- **Data Retention**: Token sesi dihapus secara otomatis saat browser ditutup (`chrome.storage.session`).

