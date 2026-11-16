# Chrome Web Store Listing — ERP Enterprise Companion

> Last Updated: 2026-11-11

---

## Store Listing

**Extension Name**
ERP Enterprise Companion

**Short Description**
Setujui dokumen ERP instan, isi formulir e-Faktur DJP otomatis, dan ekstrak mutasi rekening bank korporasi langsung dari browser.

**Detailed Description**
ERP Enterprise Companion membantu staf keuangan, pajak, perpustakaan gudang, dan manajer cabang menyelesaikan tugas harian ERP lebih cepat tanpa harus bolak-balik berpindah tab.

Fitur Utama:
Persetujuan Dokumen Instan di Side Panel: Tinjau Purchase Order, Payment Voucher, dan Journal Entry yang menunggu keputusan Anda, lalu setujui atau tolak dengan catatan langsung dari sisi layar browser.
Asisten Pengisian e-Faktur DJP: Deteksi kolom formulir faktur pajak di portal DJP Online dan isikan data NPWP, nama lawan transaksi, DPP, dan PPN dari data faktur penjualan ERP dengan satu klik untuk menghindari salah ketik.
Ekstraktor Mutasi Rekening Bank: Ambil tabel mutasi rekening dari portal internet banking perusahaan (BCA, BNI, Mandiri) ke panel kerja kas dan bank untuk mempercepat proses rekonsiliasi harian.
Penghitung Antrean Otomatis: Lencana ikon browser selalu menampilkan jumlah dokumen terbaru yang menunggu persetujuan Anda.

Keamanan & Kontrol Perusahaan:
Ekstensi ini dirancang khusus untuk organisasi pengguna Platform ERP Enterprise PT Sinergi Investasi Properti dan hanya terhubung ke gateway perusahaan yang terotorisasi. Data sesi bersifat sementara dan tidak ada kredensial login perbankan maupun data pribadi pengguna yang disimpan.

Untuk bantuan instalasi atau dukungan teknis internal, hubungi tim IT perusahaan.

**Category**
Productivity

**Single Purpose**
Menyediakan panel kerja browser untuk persetujuan dokumen ERP, pengisian formulir e-Faktur DJP, dan ekstraksi tabel mutasi rekening bank untuk rekonsiliasi kas.

**Primary Language**
Indonesian

---

## Graphics & Assets

| Asset | Dimensions | Status | Filename |
|---|---|---|---|
| Store Icon | 128×128 PNG | ✅ Ready | `icons/icon-128.png` |
| Toolbar Icon | 16×16 PNG | ✅ Ready | `icons/icon-16.png` |
| Management Icon | 48×48 PNG | ✅ Ready | `icons/icon-48.png` |
| Screenshot 1 | 1280×800 | ⬜ Not created | `screenshots/01-approvals.png` |
| Screenshot 2 | 1280×800 | ⬜ Not created | `screenshots/02-efaktur-bank.png` |
| Screenshot 3 | 1280×800 | ⬜ Not created | `screenshots/03-djp-helper.png` |
| Screenshot 4 | 1280×800 | ⬜ Not created | `screenshots/04-bank-extractor.png` |
| Small Promo Tile | 440×280 | ⬜ Not created | `screenshots/promo-small.png` |

### Screenshot Notes
1. **01-approvals.png**: Menampilkan Side Panel tab Persetujuan dengan antrean dokumen PO/Journal Entry dan tombol aksi cepat.
2. **02-efaktur-bank.png**: Menampilkan tab Pajak & Bank beserta ringkasan ekstraksi mutasi rekening.
3. **03-djp-helper.png**: Menampilkan badge biru ERP Companion aktif di situs `pajak.go.id`.
4. **04-bank-extractor.png**: Menampilkan badge hijau ERP Treasury aktif di halaman mutasi rekening bank.

---

## Permissions Justification

| Permission | Type | Justification |
|---|---|---|
| `sidePanel` | permissions | Digunakan untuk menampilkan UI primer ekstensi (antrean persetujuan cepat, asisten e-Faktur, ekstraksi mutasi bank, dan status sesi) pada browser Side Panel. |
| `storage` | permissions | Menyimpan konfigurasi URL gateway ERP (`chrome.storage.local`) serta status sinkronisasi/antrean sementara (`chrome.storage.session`). |
| `alarms` | permissions | Menjadwalkan sinkronisasi latar belakang periodik setiap 5 menit pada Service Worker MV3 untuk memperbarui badge jumlah dokumen yang menunggu persetujuan. |
| `identity` | permissions | Menjalankan autentikasi SSO OAuth2 PKCE ke Identity Provider organisasi agar akses ke Kong API Gateway menggunakan token berumur pendek tanpa menyimpan password. |
| `tabs` | permissions | Mengidentifikasi tab aktif saat pengguna menekan tombol injeksi e-Faktur atau ekstraksi bank di Side Panel agar pesan dikirim secara spesifik ke content script pada tab tujuan (`pajak.go.id` / portal bank). |
| `https://api.erp.example/*` | host_permissions | Komunikasi HTTPS terotorisasi menuju Kong API Gateway untuk mengambil antrean persetujuan, mengirim keputusan approval/reject, dan mengirim hasil ekstraksi mutasi bank ke layanan treasury. |
| `https://*.pajak.go.id/*` | host_permissions | Menginjeksikan content script `tax-helper.js` guna mendeteksi dan mengisi kolom formulir e-Faktur pada portal resmi DJP Online atas permintaan pengguna. |
| `https://*.klikbca.com/*` | host_permissions | Menginjeksikan content script `bank-extractor.js` untuk membaca tabel mutasi rekening ketika staf treasury memicu ekstraksi. |
| `https://*.bni.co.id/*` | host_permissions | Menginjeksikan content script `bank-extractor.js` untuk membaca tabel mutasi rekening BNI Corporate atas pemicuan staf treasury. |
| `https://*.bankmandiri.co.id/*` | host_permissions | Menginjeksikan content script `bank-extractor.js` untuk membaca tabel mutasi rekening Mandiri Cash Management atas pemicuan staf treasury. |
| `http://localhost:8000/*` | host_permissions | Endpoint pengembangan lokal Kong API Gateway (khusus environment development; dapat dicabut pada build produksi jika menggunakan domain `https://api.erp.example/*`). |
| `http://localhost:3000/*` | host_permissions | Endpoint pengembangan lokal aplikasi Backoffice SPA (khusus environment development). |

---

## Privacy & Data Use

### Data Collection

**Does the extension collect user data?** Yes (hanya data transaksi bisnis korporasi pada portal yang diotorisasi untuk sinkronisasi ke backend ERP perusahaan).

| Data Type | Collected? | Transmitted Off-Device? | Purpose | Shared with Third Parties? |
|---|---|---|---|---|
| Personally identifiable info | No | No | Tidak mengumpulkan nama/email pribadi konsumen | No |
| Health info | No | No | Tidak berlaku | No |
| Financial info | Yes (Mutasi rekening korporasi & rincian faktur PPN atas aksi pengguna) | Yes (Hanya ke Kong API Gateway perusahaan) | Rekonsiliasi kas/bank & pelaporan pajak e-Faktur internal | No |
| Authentication info | Yes (Token OIDC berumur pendek; **tidak pernah** PIN/password bank) | Yes (Header Bearer ke Kong API Gateway perusahaan) | Autentikasi sesi ERP pengguna internal | No |
| Personal communications | No | No | Tidak berlaku | No |
| Location | No | No | Tidak berlaku | No |
| Web history | No | No | Tidak membaca atau menyimpan riwayat browsing | No |
| User activity | No | No | Tidak ada tracking/analytics perilaku | No |
| Website content | Yes (Hanya tabel mutasi bank & struktur field e-Faktur saat dipicu) | Yes (Hasil ekstraksi ke backend ERP perusahaan) | Otomasi rekonsiliasi treasury & pengisian faktur pajak | No |

### Data Use Certification
- [x] Data is NOT sold to third parties
- [x] Data is NOT used for purposes unrelated to the extension's core functionality
- [x] Data is NOT used for creditworthiness or lending purposes
- [x] Extension does NOT use remote code (`eval` on remote scripts disallowed; untrusted HTML parsed strictly inside Manifest V3 `sandbox.html`)

---

## Privacy Policy

**Privacy Policy URL**: `https://erp.sinergi.example.com/privacy-policy`

### Ringkasan Kebijakan
1. Seluruh pertukaran data hanya terjadi antara browser pengguna dan gateway ERP internal perusahaan (`api.erp.example`) via HTTPS.
2. Kredensial perbankan (User ID, password, PIN, token fisik) **tidak pernah dibaca, disimpan, atau dikirim**.
3. Setiap tindakan persetujuan atau ekstraksi menyertakan metadata audit (`origin: chrome-extension`, `extension_version`, `X-Correlation-Id`, `X-Organization-Id`).

---

## Distribution

- **Visibility**: Private (Enterprise Private Distribution via Google Workspace Admin Console)
- **Domain Restriction**: Khusus akun organisasi `@sinergi.example.com`
- **Regions**: Indonesia & cabang operasional PT Sinergi Investasi Properti

---

## Developer Info

| Field | Value |
|---|---|
| **Publisher Name** | PT Sinergi Investasi Properti — Enterprise IT Division |
| **Contact Email** | `it-support@sinergi.example.com` |
| **Support URL** | `https://erp.sinergi.example.com/support` |
| **Homepage URL** | `https://erp.sinergi.example.com` |

---

## Version History

| Version | Date | Changes | Status |
|---|---|---|---|
| 1.0.0 | 2026-11-11 | Rilis awal Manifest V3: Side Panel Quick Approvals, e-Faktur Helper (DJP Online), Treasury Bank Statement Extractor, dan Sandbox HTML Parser. | Draft |

---

## Review Notes

### Packaging Checklist
- Jangan sertakan `CHROMEWEBSTORE.md`, `README.md`, `.git/`, atau `screenshots/` saat membuat arsip `.zip` untuk upload ke Chrome Web Store Developer Console.
- Sebelum rilis produksi ke Chrome Web Store Enterprise, hapus origin `http://localhost:*` dari `host_permissions` atau dokumentasikan jika tetap dipertahankan untuk staging internal, serta tambahkan field `"key"` di `manifest.json` bila ingin ID ekstensi konsisten dengan konfigurasi OAuth Client `chrome.identity`.
