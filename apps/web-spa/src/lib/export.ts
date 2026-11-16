export interface CsvColumn<T> {
  header: string;
  accessor: (row: T) => string | number | boolean | null | undefined;
}

/**
 * Escapes an individual CSV cell and neutralizes spreadsheet formula injection
 * while keeping valid negative financial decimals (e.g. -12500.50) unquoted as numeric.
 */
export function escapeCsvCell(value: string | number | boolean | null | undefined): string {
  if (value === undefined || value === null) return '""';
  let str = String(value);

  const isFormulaRisk =
    /^[=@\t\r]/.test(str) ||
    (/^[+-]/.test(str) && !/^[+-]?\d+(\.\d+)?$/.test(str));

  if (isFormulaRisk) {
    str = `'${str}`;
  }

  const escaped = str.replace(/"/g, '""');
  return `"${escaped}"`;
}

/**
 * Builds an RFC-4180 CSV string with UTF-8 BOM so Excel parses Unicode and separators cleanly.
 */
export function buildCsvString<T>(rows: T[], columns: CsvColumn<T>[]): string {
  const headerLine = columns.map((col) => escapeCsvCell(col.header)).join(',');
  const dataLines = rows.map((row) =>
    columns.map((col) => escapeCsvCell(col.accessor(row))).join(',')
  );
  return `\uFEFF${[headerLine, ...dataLines].join('\r\n')}`;
}

function triggerFileDownload(blob: Blob, filename: string): void {
  if (typeof document === 'undefined' || typeof URL === 'undefined') return;
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.style.display = 'none';
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

export function exportToCsv<T>(rows: T[], columns: CsvColumn<T>[], filename = 'export.csv'): void {
  const csv = buildCsvString(rows, columns);
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  triggerFileDownload(blob, filename.endsWith('.csv') ? filename : `${filename}.csv`);
}

export function exportToJson<T>(data: T, filename = 'export.json'): void {
  const json = JSON.stringify(data, null, 2);
  const blob = new Blob([json], { type: 'application/json;charset=utf-8;' });
  triggerFileDownload(blob, filename.endsWith('.json') ? filename : `${filename}.json`);
}

export function triggerPrint(): void {
  if (typeof window !== 'undefined' && typeof window.print === 'function') {
    window.print();
  }
}
