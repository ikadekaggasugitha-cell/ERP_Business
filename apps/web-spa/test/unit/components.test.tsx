import * as React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { I18nextProvider } from 'react-i18next';
import { describe, expect, it, vi } from 'vitest';
import { VirtualizedTable, type ColumnDef } from '../../src/components/data-table/VirtualizedTable';
import { AppDialog } from '../../src/components/ui/AppDialog';
import { BrandLogo } from '../../src/components/ui/BrandLogo';
import { Button } from '../../src/components/ui/Button';
import { DomainErrorBoundary } from '../../src/components/ui/DomainErrorBoundary';
import { ErrorState } from '../../src/components/ui/ErrorState';
import { KpiCard } from '../../src/components/ui/KpiCard';
import { MoneyInput } from '../../src/components/ui/MoneyInput';
import { SideDrawer } from '../../src/components/ui/SideDrawer';
import { StatusBadge } from '../../src/components/ui/StatusBadge';
import i18n from '../../src/lib/i18n';

function wrapWithProviders(ui: React.ReactNode) {
  return render(<I18nextProvider i18n={i18n}>{ui}</I18nextProvider>);
}

describe('4-Pillar UI Components', () => {
  it('Button disables during loading state and invokes click when ready', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();

    const { rerender } = wrapWithProviders(<Button onClick={onClick}>Simpan</Button>);
    await user.click(screen.getByRole('button', { name: 'Simpan' }));
    expect(onClick).toHaveBeenCalledTimes(1);

    rerender(
      <I18nextProvider i18n={i18n}>
        <Button isLoading onClick={onClick}>
          Simpan
        </Button>
      </I18nextProvider>
    );
    expect(screen.getByRole('button', { name: 'Simpan' })).toBeDisabled();
  });

  it('MoneyInput preserves string arithmetic formatting on blur', () => {
    let latest = '1000';
    const handleChange = vi.fn((next: string) => {
      latest = next;
    });

    const { rerender } = wrapWithProviders(
      <MoneyInput value={latest} onValueChange={handleChange} currency="IDR" precision={2} />
    );
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: '25000.5abc' } });
    expect(handleChange).toHaveBeenLastCalledWith('25000.5');

    rerender(
      <I18nextProvider i18n={i18n}>
        <MoneyInput value="25000.5" onValueChange={handleChange} currency="IDR" precision={2} />
      </I18nextProvider>
    );
    fireEvent.blur(screen.getByRole('textbox'));
    expect(handleChange).toHaveBeenLastCalledWith('25000.50');
  });

  it('AppDialog renders accessible title, body, footer, and closes on Escape', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => { });

    wrapWithProviders(
      <AppDialog
        open={true}
        onOpenChange={onOpenChange}
        title="Buat Jurnal Baru"
        footer={<button type="button">Simpan Draft</button>}
      >
        <div>Form Content</div>
      </AppDialog>
    );

    const dialog = screen.getByRole('dialog', { name: 'Buat Jurnal Baru' });
    expect(dialog).toBeInTheDocument();
    expect(dialog).not.toHaveAttribute('aria-describedby');
    expect(screen.getByText('Buat Jurnal Baru')).toBeInTheDocument();
    expect(screen.getByText('Form Content')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Simpan Draft' })).toBeInTheDocument();
    expect(warnSpy).not.toHaveBeenCalled();

    await user.keyboard('{Escape}');
    expect(onOpenChange).toHaveBeenCalledWith(false);
    warnSpy.mockRestore();
  });

  it('AppDialog binds aria-describedby only when description is non-empty', () => {
    const { rerender } = wrapWithProviders(
      <AppDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Konfirmasi Posting"
        description="Jurnal akan diposting ke buku besar."
      >
        <div>Body</div>
      </AppDialog>
    );

    const describedDialog = screen.getByRole('dialog', { name: 'Konfirmasi Posting' });
    const descId = describedDialog.getAttribute('aria-describedby');
    expect(descId).toBeTruthy();
    const descEl = document.getElementById(descId!);
    expect(descEl).toHaveTextContent('Jurnal akan diposting ke buku besar.');

    rerender(
      <I18nextProvider i18n={i18n}>
        <AppDialog open={true} onOpenChange={vi.fn()} title="Konfirmasi Posting" description="   ">
          <div>Body</div>
        </AppDialog>
      </I18nextProvider>
    );
    expect(screen.getByRole('dialog', { name: 'Konfirmasi Posting' })).not.toHaveAttribute(
      'aria-describedby'
    );
  });

  it('SideDrawer renders title once without duplicate description, supports description/footer, and closes on Escape', async () => {
    const user = userEvent.setup();
    const onDrawerOpenChange = vi.fn();
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => { });

    const { rerender } = wrapWithProviders(
      <SideDrawer
        open={true}
        onOpenChange={onDrawerOpenChange}
        title="Detail Faktur"
        footer={<button type="button">Tutup</button>}
      >
        <div>Drawer Body</div>
      </SideDrawer>
    );

    const drawerWithoutDesc = screen.getByRole('dialog', { name: 'Detail Faktur' });
    expect(drawerWithoutDesc).not.toHaveAttribute('aria-describedby');
    expect(screen.getByText('Detail Faktur')).toBeInTheDocument();
    expect(screen.getByText('Drawer Body')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Tutup' })).toBeInTheDocument();
    expect(warnSpy).not.toHaveBeenCalled();

    await user.keyboard('{Escape}');
    expect(onDrawerOpenChange).toHaveBeenCalledWith(false);

    rerender(
      <I18nextProvider i18n={i18n}>
        <SideDrawer
          open={true}
          onOpenChange={onDrawerOpenChange}
          title="Detail Faktur"
          description="Ringkasan baris faktur vendor."
        >
          <div>Drawer Body</div>
        </SideDrawer>
      </I18nextProvider>
    );
    const drawerWithDesc = screen.getByRole('dialog', { name: 'Detail Faktur' });
    const drawerDescId = drawerWithDesc.getAttribute('aria-describedby');
    expect(drawerDescId).toBeTruthy();
    expect(document.getElementById(drawerDescId!)).toHaveTextContent(
      'Ringkasan baris faktur vendor.'
    );
    warnSpy.mockRestore();
  });

  it('VirtualizedTable renders rows, handles clicks, and shows empty state', async () => {
    const user = userEvent.setup();
    const onRowClick = vi.fn();

    interface Row {
      id: string;
      code: string;
      name: string;
    }

    const columns: ColumnDef<Row>[] = [
      { header: 'Code', accessorKey: 'code' },
      { header: 'Name', accessorKey: 'name' },
    ];

    const rows: Row[] = [
      { id: '1', code: 'JV-001', name: 'Posting Kas' },
      { id: '2', code: 'JV-002', name: 'Biaya Sewa' },
    ];

    const onPageChange = vi.fn();
    const { rerender } = wrapWithProviders(
      <VirtualizedTable
        data={rows}
        columns={columns}
        onRowClick={onRowClick}
        pagination={{
          page: 1,
          size: 25,
          totalElements: 60,
          totalPages: 3,
          onPageChange,
        }}
      />
    );

    expect(screen.getByText('JV-001')).toBeInTheDocument();
    await user.click(screen.getByText('JV-001'));
    expect(onRowClick).toHaveBeenCalledWith(rows[0]);

    const grid = screen.getByRole('grid');
    fireEvent.keyDown(grid, { key: 'ArrowDown' });
    fireEvent.keyDown(grid, { key: 'Enter' });
    expect(onRowClick).toHaveBeenCalledTimes(2);

    const prevBtn = screen.getByRole('button', { name: /Sebelumnya|Previous/i });
    const nextBtn = screen.getByRole('button', { name: /Berikutnya|Next/i });
    expect(prevBtn).toBeDisabled();
    expect(nextBtn).toBeEnabled();
    await user.click(nextBtn);
    expect(onPageChange).toHaveBeenCalledWith(2);

    rerender(
      <I18nextProvider i18n={i18n}>
        <VirtualizedTable data={[]} columns={columns} />
      </I18nextProvider>
    );
    expect(screen.getByText(/Belum Ada Data|No Records Found/i)).toBeInTheDocument();
  });

  it('ErrorState triggers retry handler and DomainErrorBoundary isolates child crashes', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();

    wrapWithProviders(
      <ErrorState code="GL_500" message="Gagal memuat" correlationId="cid-123456" onRetry={onRetry} />
    );
    await user.click(screen.getByRole('button', { name: /Coba Lagi|Retry/i }));
    expect(onRetry).toHaveBeenCalledTimes(1);

    const spy = vi.spyOn(console, 'error').mockImplementation(() => { });
    const Crash = (): React.ReactNode => {
      throw new Error('Isolated domain failure');
    };
    wrapWithProviders(
      <DomainErrorBoundary resetKey="test">
        <Crash />
      </DomainErrorBoundary>
    );
    expect(screen.getByText('Isolated domain failure')).toBeInTheDocument();
    spy.mockRestore();
  });

  it('supports BrandLogo, Button danger-soft, KpiCard badge, SideDrawer titleBadge, and VirtualizedTable selectedRowId', () => {
    const { container, rerender } = wrapWithProviders(
      <div>
        <BrandLogo size="md" />
        <Button variant="danger-soft">Balik Jurnal</Button>
        <KpiCard
          label="Total Aset"
          value="Rp 48,25 M"
          badge="3 Posted"
          badgeTone="info"
          subtitle="Live GL"
        />
        <VirtualizedTable
          data={[
            { id: 'r1', code: 'JRN-01' },
            { id: 'r2', code: 'JRN-02' },
          ]}
          columns={[{ header: 'Kode', accessorKey: 'code' }]}
          selectedRowId="r1"
        />
      </div>
    );

    expect(container.querySelector('svg')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Balik Jurnal' }).className).toContain('bg-rose-50');
    expect(screen.getByText('3 Posted')).toBeInTheDocument();

    const selectedRow = screen.getByText('JRN-01').closest('[role="row"]');
    expect(selectedRow).toHaveAttribute('aria-selected', 'true');

    rerender(
      <I18nextProvider i18n={i18n}>
        <SideDrawer
          open={true}
          onOpenChange={vi.fn()}
          title="JRN-2026-0842"
          titleBadge={<StatusBadge status="POSTED" />}
        >
          <div>Detail Body</div>
        </SideDrawer>
      </I18nextProvider>
    );
    expect(screen.getByText('Diposting')).toBeInTheDocument();
  });
});
