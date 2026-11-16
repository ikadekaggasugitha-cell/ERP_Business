# ADR-012: Enterprise Chrome Extension Companion (Manifest V3)

**Status**: Accepted  
**Tanggal**: 2026-11-11  
**Pengambil Keputusan**: Solution Architect, Lead Security Engineer, Product Lead  
**Terkait**: ADR-003 (Kong API Gateway), ADR-007 (API Style: REST), INT-005 (Bank/Payment Gateway), INT-006 (e-Faktur / Pajak), NFR-016 (Security Management), NFR-019 (Compliance & Data Protection)

---

## Konteks

Operasional bisnis harian pada perusahaan berskala enterprise melibatkan interaksi terus-menerus antara staf ERP dengan berbagai portal web pihak ketiga dan perangkat keras eksternal:
1. **Portal Otoritas Pajak & Kepatuhan Pemerintah**: Staf perpajakan harus menerbitkan atau memvalidasi faktur pajak elektronik di portal pemerintah (misalnya DJP Online / e-Faktur / CoreTax di Indonesia, IRAS di Singapura, GSTN di India, serta jaringan e-invoicing PEPPOL di Eropa). Proses manual dengan copy-paste atau ekspor-impor file CSV rentan terhadap keterlambatan, rekonsiliasi yang macet, dan salah input.
2. **Portal Perbankan Korporasi (Internet Banking B2B)**: Staf perbendaharaan (treasury) mengakses rekening koran elektronik harian untuk rekonsiliasi kas/bank.
3. **Persetujuan Cepat Manajer/Eksekutif (*In-Context Workflow Approvals*)**: Eksekutif dan manajer cabang sering kali berada di luar tab ERP (sedang membaca email, dokumen kerja, atau spreadsheet). Mereka membutuhkan cara cepat melihat notifikasi mendesak dan menyetujui Purchase Order, Journal Entry, atau Payroll Run tanpa harus membuka tab ERP penuh dan mencari menu approval.
4. **Integrasi Perangkat Keras Gudang & POS**: Barcode scanner USB/Bluetooth, RFID reader, dan timbangan digital sering kali memerlukan driver desktop berat jika ingin berkomunikasi dengan web ERP.

Dibutuhkan ekstensi browser resmi (*Enterprise Companion Extension*) yang aman, ringan, dan tersertifikasi untuk menjembatani kebutuhan ini secara *seamless*.

---

## Keputusan

Kami menetapkan pengembangan **Enterprise Chrome Extension Companion** resmi berbasis standar modern **Google Chrome Manifest V3**:

1. **Manifest V3 Core Architecture**:
   - **Background Service Worker**: Bersifat *ephemeral* (non-persistent). State tidak disimpan pada variabel global, melainkan dipersistensikan pada `chrome.storage.session` (terisolasi dalam sesi browser aktif) atau `chrome.storage.local`. Menggunakan `chrome.alarms` untuk penjadwalan periodik (bukan `setInterval`).
   - **Side Panel Interface (`chrome.sidePanel`)**: Menyediakan panel sisi interaktif untuk ringkasan metrik harian, daftar antrean approval dokumen, dan aksi one-click (Approve/Reject dengan catatan) tanpa mengganggu tab aktif pengguna.
   - **Content Scripts Terisolasi**: Injeksi skrip hanya pada domain terverifikasi (*strictly scoped host permissions*, misalnya domain resmi DJP / portal bank yang disetujui organisasi) untuk membaca status faktur atau mengotomasi validasi kode QR.
   - **Sandboxed Execution Context**: Pemrosesan dokumen pihak ketiga, ekspresi reguler, atau parsing data HTML eksternal dijalankan dalam *sandboxed iframe* (`sandbox.html`) yang menerapkan *strict Content Security Policy (CSP)* dan komunikasi via `postMessage`.

2. **Fungsi Utama Pendukung ERP**:
   - **In-Context Tax & E-Invoicing Helper**: Mengambil data faktur yang disetujui dari ERP melalui Kong API Gateway (berdasarkan token user aktif), memvalidasi nomor seri faktur pajak, dan membantu injeksi data ke portal pajak resmi, lalu menarik kembali kode NTTE / QR validasi ke basis data ERP.
   - **Treasury Bank Statement Extractor**: Membantu staf treasury mengekstraksi mutasi rekening dari sesi portal bank yang sah dan mengirimkannya ke `treasury-cashbank-service` untuk pencocokan otomatis (*auto-reconciliation*).
   - **Hardware Web-Bridge**: Memanfaatkan Web Serial API / Web HID API dalam konteks browser yang aman untuk menerima stream input dari barcode scanner, handheld terminal, atau RFID reader langsung ke form ERP.

3. **Keamanan, Token & Izin (Zero-Trust Security)**:
   - **Autentikasi OAuth2 PKCE via `chrome.identity`**: Token akses berumur pendek (*short-lived JWT*) diperoleh via flow PKCE terintegrasi dengan Identity Provider organisasi.
   - **Tidak Ada Penyimpanan Kredensial Permanen**: Password atau private key tidak pernah disimpan di storage ekstensi.
   - **Scoped Host Permissions**: Ekstensi dilarang menggunakan izin wildcard `<all_urls>`. Host permissions dibatasi ketat per whitelist organisasi.
   - **Audit Trail Seluruh Aksi Ekstensi**: Setiap aksi approval atau injeksi data dari ekstensi mencantumkan metadata `origin: chrome-extension` dan `extension_version` pada payload audit log ERP.

4. **Tata Kelola Distribusi & Metadata Toko**:
   - Seluruh metadata rilis, justifikasi izin granular, dan kebijakan privasi didokumentasikan dalam artefak `CHROMEWEBSTORE.md` di repositori, serta dikontrol melalui Google Chrome Web Store Enterprise Private Distribution (khusus domain/karyawan organisasi).

---

## Rasional

- Mengeliminasi *human error* pada proses kritikal pelaporan pajak dan rekonsiliasi kasir/bank antar-cabang.
- Mempercepat siklus persetujuan dokumen (*approval cycle time*) dari hitungan jam/hari menjadi hitungan menit melalui integrasi Side Panel.
- Standar Manifest V3 menjamin konsumsi memori dan CPU browser tetap rendah, aman dari injeksi skrip jahat, dan mematuhi kebijakan keamanan Google Chrome terbaru.

---

## Konsekuensi

### Positif (+)
- Efisiensi staf operasional dan kepatuhan regulasi perpajakan meningkat signifikan.
- Pengalaman persetujuan workflow instan bagi level manajemen.
- Terkoneksi aman dengan ekosistem backend via Kong API Gateway dan audit trail penuh.

### Negatif (−)
- Memerlukan pemeliharaan rutin jika tata letak atau DOM portal web eksternal (pajak/bank) mengalami pembaruan struktural.
- Siklus rilis ekstensi harus mengikuti proses review Google Web Store atau Enterprise Policy Deploy.

