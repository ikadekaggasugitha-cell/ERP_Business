# ADR-014: Tri-Currency & Multi-Country Real-Time Accounting Engine

**Status**: Accepted  
**Tanggal**: 2026-11-11  
**Pengambil Keputusan**: Chief Financial Officer, Lead Financial Architect, General Ledger Lead  
**Terkait**: ADR-001 (Microservices & DB-per-service), ADR-005 (Selective Exactly-Once), BRD BAB 9.18 (Currency Management Master), BRD BAB 12.19 (Localization & Multi-Language), PSAK 10 / IAS 21

---

## Konteks

Platform ERP dirancang untuk mendukung holding enterprise internasional dengan anak perusahaan, cabang, dan kantor perwakilan yang tersebar di berbagai negara (misalnya Indonesia, Singapura, Malaysia, Jepang, dan Eropa). Setiap entitas beroperasi di bawah rezim regulasi, mata uang statutori, dan kalender fiskal yang berbeda.

Tantangan utama yang dihadapi pada skala jutaan transaksi:
1. **Kompleksitas Multi-Valuta**: Transaksi pembelian/penjualan sering kali menggunakan valuta asing (mis. USD), sementara pembukuan statutori cabang menggunakan mata uang lokal (mis. IDR atau SGD), dan laporan konsolidasi grup holding menggunakan mata uang pelaporan grup (mis. IDR atau USD). Menghitung ulang kurs pada saat laporan ditarik akan memperlambat query secara dramatis (*heavy on-the-fly currency conversion*).
2. **Volatilitas Kurs & Selisih Kurs (FX Gain/Loss)**: Fluktuasi nilai tukar mengharuskan pencatatan *realized exchange gain/loss* saat pelunasan piutang/utang, serta *unrealized exchange gain/loss* pada akhir bulan (*closing period*) sesuai standar akuntansi internasional **PSAK 10 / IAS 21**.
3. **Perbedaan Kalender Fiskal Statutori**: Cabang di Jepang atau India menggunakan tahun fiskal 1 April – 31 Maret, cabang di Australia menggunakan 1 Juli – 30 Juni, sedangkan entitas di Indonesia dan AS menggunakan 1 Januari – 31 Desember. Ada pula cabang ritel yang menggunakan kalender 4-4-5 minggu.
4. **Presisi Finansial Tanpa Pembulatan Pecah**: Penggunaan tipe data floating-point dapat menimbulkan selisih sen (*penny rounding error*) yang merusak integritas neraca saldo (*trial balance*).

---

## Keputusan

Kami menetapkan arsitektur **Tri-Currency & Multi-Country Real-Time Accounting Engine** sebagai fondasi transaksi finansial di seluruh layanan:

1. **Arsitektur Tri-Currency pada Setiap Baris Jurnal & Faktur**:
   Setiap baris transaksi finansial (`gl.journal_line`, `ar.invoice_line`, `ap.invoice_line`, `treasury.cash_transaction`) wajib menyimpan 3 pasang nilai secara terdenormalisasi pada saat penulisan (*write-time*):
   - **Transaction Currency (`tc`)**: Nilai dalam mata uang yang disepakati pada transaksi asli (mis. USD 10,000.00).
   - **Functional Currency (`fc`)**: Nilai dalam mata uang statutori entitas hukum pemilik transaksi (mis. IDR 160.000.000,00) beserta nilai kurs yang berlaku (`exchange_rate_fc`).
   - **Presentation Currency (`pc`)**: Nilai dalam mata uang standar konsolidasi holding (mis. USD 10,000.00 atau IDR 160.000.000,00) beserta kurs konsolidasi (`exchange_rate_pc`).

   *Dampak*: Laporan keuangan lokal dan neraca konsolidasi holding dapat ditarik langsung dalam milidetik tanpa konversi kurs saat query baca.

2. **Mesin Triangulasi Kurs Real-Time Berbasis Cache**:
   - `currency-service` memelihara tabel kurs harian (Spot Rate, Pajak/KMK, Average Rate, Closing Rate).
   - Seluruh kurs aktif di-*cache* dalam memori / Redis dengan TTL 1 jam untuk melayani verifikasi ribuan transaksi per detik tanpa beban ke basis data.
   - Jika pasangan mata uang langsung (mis. IDR ke SEK) tidak tersedia, mesin secara otomatis melakukan **triangulasi** melalui mata uang jangkar (*anchor currency*, USD):
     $$\text{Kurs IDR/SEK} = \frac{\text{Kurs USD/SEK}}{\text{Kurs USD/IDR}}$$
   - Nilai kurs disimpan dengan presisi tinggi: `numeric(18,8)` untuk kurs, dan `numeric(20,4)` untuk nilai moneter.

3. **Mesin Otomasi Selisih Kurs (FX Revaluation - PSAK 10 / IAS 21)**:
   - **Realized Gain/Loss**: Dihitung dan diposting otomatis ke GL saat pelunasan invoice (perbedaan kurs antara tanggal pengakuan faktur dan tanggal pembayaran kas/bank).
   - **Unrealized Gain/Loss (Month-End)**: Job berkala membaca saldo pos moneter terbuka (piutang, utang, kas/bank valas) per tanggal penutupan buku, menghitung selisih terhadap kurs penutupan (*closing rate*), dan menerbitkan jurnal penyesuaian otomatis yang berstatus *auto-reversing* pada awal periode berikutnya.

4. **Mesin Kalender Fiskal Multi-Negara (*Dual-Period Tagging*)**:
   - Setiap entitas organisasi (`organization_id`) terikat pada kalender fiskal statutori lokalnya.
   - Pada setiap baris transaksi dicatat dua penanda periode:
     - `local_fiscal_period_id`: Menentukan periode pembukuan resmi cabang (mis. Periode 1 = April untuk entitas Jepang).
     - `group_fiscal_period_id`: Menentukan periode konsolidasi holding grup (mis. Periode 4 = April untuk holding Indonesia).
   - Mendukung kalender reguler 12 bulan, periode ke-13/14 untuk audit closing, serta kalender ritel 4-4-5.

---

## Rasional

- Menjamin konsistensi neraca holding dan cabang tanpa perlu rekonsiliasi manual antar mata uang di akhir bulan.
- Memenuhi kepatuhan penuh terhadap standar PSAK 10, IAS 21, dan regulasi perpajakan lokal (kurs KMK/DJP di Indonesia, MAS di Singapura).
- Memungkinkan pembacaan laporan konsolidasi holding jutaan transaksi secara instan karena nilai *presentation currency* telah tersimpan pada baris jurnal.

---

## Konsekuensi

### Positif (+)
- Performa pelaporan keuangan konsolidasi holding berkecepatan tinggi (NFR-003).
- Kepatuhan audit statutori multi-negara terjamin dan terhindar dari sanksi pajak/akuntansi.
- Selisih pembulatan moneter dieliminasi sepenuhnya dengan tipe `numeric(20,4)`.

### Negatif (−)
- Ukuran baris basis data tabel finansial sedikit bertambah karena menyimpan 3 pasang nilai moneter.
- Layanan transaksi finansial bergantung pada ketersediaan kurs aktif dari `currency-service` (dimitigasi dengan caching in-memory dan fallback rate).

