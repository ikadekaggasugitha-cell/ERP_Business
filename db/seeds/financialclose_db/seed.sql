-- Seed: financialclose_db (reference). Idempotent. Run: psql -d financialclose_db -f this
-- Optional: -v org_id=<uuid> -v actor=<uuid> to seed for a specific tenant.
\set ON_ERROR_STOP on
\if :{?org_id} \else \set org_id '00000000-0000-0000-0000-000000000000' \endif
\if :{?actor}  \else \set actor  '00000000-0000-0000-0000-000000000001' \endif

INSERT INTO fcl.closing_checklist_template (id, organization_id, code, name, financial_organization_ref, created_by) VALUES
  ('f0c10000-0000-4000-8000-000000000001', :'org_id','ACC-RECON','Rekonsiliasi rekening bank',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000002', :'org_id','AR-AGING','Review umur piutang dan estimasi ECL',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000003', :'org_id','AP-ACCRUE','Akrual beban dan biaya yang terutang',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000004', :'org_id','INV-COUNT','Rekonsiliasi persediaan dan opname',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000005', :'org_id','FA-DEPR','Posting penyusutan aset tetap periode berjalan',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000006', :'org_id','TAX-CORP','Perhitungan PPh korporasi periode berjalan',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000007', :'org_id','FX-REVAL','Revaluasi selisih kurs saldo valas (PSAK 10 / IAS 21)',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000008', :'org_id','JE-CLOSE','Entri jurnal penutup dan review posting',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000009', :'org_id','TB-RECON','Rekonsiliasi neraca saldo dan buku besar',:'org_id',:'actor'),
  ('f0c10000-0000-4000-8000-000000000010', :'org_id','REP-PREP','Penyusunan laporan keuangan interim',:'org_id',:'actor')
ON CONFLICT (organization_id, code) DO NOTHING;
