# Analisis Kebutuhan Bisnis ERP

## 1. Latar Belakang

ERP dirancang dengan fokus pada **fleksibilitas struktur organisasi**
sehingga mampu mendukung perusahaan tunggal hingga grup perusahaan
(Holding) tanpa perubahan source code.

## 2. Tujuan

-   Mendukung Single Company, Holding, Sub Holding, Anak Perusahaan,
    SBU, Cabang, Gudang, dan Project Site.
-   Mendukung ekspansi organisasi melalui konfigurasi.

## 3. Kebutuhan Inti

### 3.1 Multi Organization

Struktur organisasi bersifat hierarkis dan tidak dibatasi jumlah level.

``` text
Holding
├── Sub Holding A
│   ├── Company A1
│   └── Company A2
└── Company B
```

### 3.2 Unlimited Organization Level

Contoh level: - Holding - Sub Holding - Business Group - Company -
Division - Branch - Department - Section - Team

### 3.3 Organization Tree

Setiap organisasi memiliki parent sehingga membentuk pohon organisasi.

### 3.4 Shared Master Data

Master data tertentu dapat digunakan bersama (Vendor, Currency, Tax).

### 3.5 Isolated Master Data

Master tertentu bersifat lokal (COA, Customer bila diperlukan).

### 3.6 Configurable Sharing

Scope master data: - Global - Holding - Sub Holding - Company - Branch

### 3.7 Scope-Based Data Access

Hak akses mengikuti scope organisasi. - CEO Holding → seluruh data -
Direktur Company → company dan turunannya - Manager Branch → branch
terkait - Staff → unit masing-masing

### 3.8 Flexible Role

Role dapat dipasang pada level organisasi mana pun.

### 3.9 Configurable Workflow

Alur approval dibangun melalui konfigurasi, bukan hardcode.

### 3.10 Modular ERP

Core, HR, Finance, Inventory, Procurement, CRM, Sales, Manufacturing,
Asset, Project, Maintenance, Payroll, dll.

### 3.11 Multi Currency

Mendukung mata uang dasar berbeda per company dan konsolidasi holding.

### 3.12 Multi Language

Mendukung berbagai bahasa tanpa perubahan kode.

### 3.13 Audit Trail

Menyimpan pengguna, organisasi, waktu, perubahan, dan alasan perubahan.

## 4. Prinsip Arsitektur Bisnis

1.  Configuration over Customization.
2.  Organization Tree First.
3.  Scope-Based Everything.
4.  Modular & Extensible.
5.  Multi-Tenant Ready.

## 5. Kesimpulan

ERP dibangun sebagai platform enterprise yang dapat berkembang dari satu
perusahaan menjadi grup perusahaan besar hanya melalui konfigurasi
organisasi, hak akses, workflow, dan scope data.
