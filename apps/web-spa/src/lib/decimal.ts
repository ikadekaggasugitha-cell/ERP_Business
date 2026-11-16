import type { AppLocale } from '../types';

export type SupportedLocale = AppLocale | 'id-ID' | 'en-US';

function resolveIntlLocale(locale?: SupportedLocale): 'id-ID' | 'en-US' {
  return locale === 'en' || locale === 'en-US' ? 'en-US' : 'id-ID';
}

function normalizeDecimalInput(raw: string | number | null | undefined): {
  isNeg: boolean;
  intPart: string;
  fracPart: string;
} {
  if (raw === undefined || raw === null) {
    return { isNeg: false, intPart: '0', fracPart: '' };
  }
  const clean = String(raw).trim();
  if (!clean || clean === 'NaN' || clean === 'Infinity' || clean === '-Infinity' || clean === '-' || clean === '+') {
    return { isNeg: false, intPart: '0', fracPart: '' };
  }

  const sciMatch = clean.match(/^([+-]?)(\d+)(?:\.(\d+))?[eE]([+-]?\d+)$/);
  if (sciMatch) {
    const isNeg = sciMatch[1] === '-';
    const intDigits = sciMatch[2] ?? '0';
    const fracDigits = sciMatch[3] ?? '';
    const exp = Number.parseInt(sciMatch[4] ?? '0', 10);
    const combined = intDigits + fracDigits;
    const decimalIdx = intDigits.length + exp;

    let resInt = '0';
    let resFrac = '';
    if (decimalIdx <= 0) {
      resInt = '0';
      resFrac = '0'.repeat(-decimalIdx) + combined;
    } else if (decimalIdx >= combined.length) {
      resInt = combined + '0'.repeat(decimalIdx - combined.length);
      resFrac = '';
    } else {
      resInt = combined.slice(0, decimalIdx);
      resFrac = combined.slice(decimalIdx);
    }

    return {
      isNeg,
      intPart: resInt.replace(/^0+(?=\d)/, '') || '0',
      fracPart: resFrac,
    };
  }

  const isNeg = clean.startsWith('-');
  const unsigned = clean.replace(/^[+-]/, '');
  const [rawInt = '0', rawFrac = ''] = unsigned.split('.');
  const intDigits = (rawInt.replace(/\D/g, '') || '0').replace(/^0+(?=\d)/, '') || '0';
  const fracDigits = rawFrac.replace(/\D/g, '');
  return { isNeg, intPart: intDigits, fracPart: fracDigits };
}

function toScaledBigInt(
  raw: string | number | Decimal | null | undefined,
  targetScale: number
): bigint {
  const str = raw instanceof Decimal ? raw.toRawString() : raw;
  const norm = normalizeDecimalInput(str);
  const frac =
    norm.fracPart.length >= targetScale
      ? norm.fracPart.slice(0, targetScale)
      : norm.fracPart.padEnd(targetScale, '0');
  const magnitude = BigInt(norm.intPart + frac);
  if (magnitude === 0n) return 0n;
  return norm.isNeg ? -magnitude : magnitude;
}

function formatScaledBigInt(value: bigint, scale: number, resultDecimals: number): string {
  if (value === 0n) {
    return resultDecimals > 0 ? `0.${'0'.repeat(resultDecimals)}` : '0';
  }
  const isNeg = value < 0n;
  const absVal = isNeg ? -value : value;

  let str = absVal.toString();
  if (str.length <= scale) {
    str = str.padStart(scale + 1, '0');
  }

  const splitIdx = str.length - scale;
  const wholePart = str.slice(0, splitIdx) || '0';
  let fracPart = str.slice(splitIdx);

  if (resultDecimals >= 0) {
    if (fracPart.length > resultDecimals) {
      const roundDigitChar = fracPart[resultDecimals] ?? '0';
      const roundDigit = roundDigitChar >= '0' && roundDigitChar <= '9' ? Number(roundDigitChar) : 0;
      fracPart = fracPart.slice(0, resultDecimals);
      if (roundDigit >= 5) {
        const fullShifted = BigInt(wholePart + fracPart) + 1n;
        const shiftedStr = fullShifted.toString().padStart(resultDecimals + 1, '0');
        const rWhole = shiftedStr.slice(0, shiftedStr.length - resultDecimals) || '0';
        const rFrac = shiftedStr.slice(shiftedStr.length - resultDecimals);
        return (isNeg ? '-' : '') + (resultDecimals > 0 ? `${rWhole}.${rFrac}` : rWhole);
      }
    } else {
      fracPart = fracPart.padEnd(resultDecimals, '0');
    }
  }

  if (wholePart === '0' && /^0*$/.test(fracPart)) {
    return resultDecimals > 0 ? `0.${'0'.repeat(resultDecimals)}` : '0';
  }

  return (isNeg ? '-' : '') + (fracPart ? `${wholePart}.${fracPart}` : wholePart);
}

/**
 * Exact decimal string addition without IEEE-754 floating point precision loss.
 */
export function addDecimals(aStr: string, bStr: string, resultDecimals = 2): string {
  const a = normalizeDecimalInput(aStr);
  const b = normalizeDecimalInput(bStr);

  const scale = Math.max(a.fracPart.length, b.fracPart.length, resultDecimals >= 0 ? resultDecimals : 0, 4);
  const bigA = BigInt(a.intPart + a.fracPart.padEnd(scale, '0')) * (a.isNeg ? -1n : 1n);
  const bigB = BigInt(b.intPart + b.fracPart.padEnd(scale, '0')) * (b.isNeg ? -1n : 1n);

  return formatScaledBigInt(bigA + bigB, scale, resultDecimals);
}

/**
 * Exact decimal string subtraction without IEEE-754 floating point precision loss.
 */
export function subtractDecimals(aStr: string, bStr: string, resultDecimals = 2): string {
  const b = normalizeDecimalInput(bStr);
  const toggledB = (b.isNeg ? '' : '-') + `${b.intPart}.${b.fracPart}`;
  return addDecimals(aStr, toggledB, resultDecimals);
}

/**
 * Exact decimal string multiplication without IEEE-754 floating point precision loss.
 * Uses BigInt integer scaling to guarantee enterprise accounting precision (NUMERIC(20,4)).
 */
export function multiplyDecimals(aStr: string, bStr: string, resultDecimals = 4): string {
  const defZero = resultDecimals > 0 ? `0.${'0'.repeat(resultDecimals)}` : '0';
  if (!aStr || !bStr) return defZero;

  const a = normalizeDecimalInput(aStr);
  const b = normalizeDecimalInput(bStr);

  const totalScale = a.fracPart.length + b.fracPart.length;
  const bigA = BigInt(a.intPart + a.fracPart);
  const bigB = BigInt(b.intPart + b.fracPart);
  const product = bigA * bigB;

  if (product === 0n) return defZero;

  const isNeg = a.isNeg !== b.isNeg;
  return formatScaledBigInt(isNeg ? -product : product, totalScale, resultDecimals);
}

/**
 * Exact BigInt fixed-point division with half-up rounding and zero-divisor protection.
 */
export function divideDecimals(aStr: string, bStr: string, resultDecimals = 4): string {
  const a = normalizeDecimalInput(aStr);
  const b = normalizeDecimalInput(bStr);

  const commonScale = Math.max(a.fracPart.length, b.fracPart.length);
  const bigA = BigInt(a.intPart + a.fracPart.padEnd(commonScale, '0'));
  const bigB = BigInt(b.intPart + b.fracPart.padEnd(commonScale, '0'));

  if (bigB === 0n) {
    throw new Error('Division by zero in divideDecimals');
  }

  const defZero = resultDecimals > 0 ? `0.${'0'.repeat(resultDecimals)}` : '0';
  if (bigA === 0n) return defZero;

  const precision = Math.max(resultDecimals, 0);
  // Compute one extra decimal digit for exact half-up rounding
  const factor = 10n ** BigInt(precision + 1);
  const rawQuotient = (bigA * factor) / bigB;
  const lastDigit = rawQuotient % 10n;
  let rounded = rawQuotient / 10n;
  if (lastDigit >= 5n) {
    rounded += 1n;
  }

  const isNeg = a.isNeg !== b.isNeg;
  return formatScaledBigInt(isNeg ? -rounded : rounded, precision, precision);
}

/**
 * Immutable Fixed-Point Decimal helper class for SPA financial aggregations.
 */
export class Decimal {
  private readonly raw: string;
  private readonly defaultDecimals: number;

  constructor(value: string | number | Decimal = '0', defaultDecimals?: number) {
    if (value instanceof Decimal) {
      this.raw = value.raw;
      this.defaultDecimals = defaultDecimals ?? value.defaultDecimals;
    } else {
      const norm = normalizeDecimalInput(value);
      const frac = norm.fracPart || '00';
      const isZero = norm.intPart === '0' && /^0+$/.test(frac);
      this.raw = `${!isZero && norm.isNeg ? '-' : ''}${norm.intPart}.${frac}`;
      this.defaultDecimals = defaultDecimals ?? Math.max(norm.fracPart.length, 2);
    }
  }

  add(other: string | number | Decimal, decimals = Math.max(this.defaultDecimals, 4)): Decimal {
    const otherStr = other instanceof Decimal ? other.raw : String(other);
    return new Decimal(addDecimals(this.raw, otherStr, decimals), decimals);
  }

  sub(other: string | number | Decimal, decimals = Math.max(this.defaultDecimals, 4)): Decimal {
    const otherStr = other instanceof Decimal ? other.raw : String(other);
    return new Decimal(subtractDecimals(this.raw, otherStr, decimals), decimals);
  }

  mul(other: string | number | Decimal, decimals = Math.max(this.defaultDecimals, 4)): Decimal {
    const otherStr = other instanceof Decimal ? other.raw : String(other);
    return new Decimal(multiplyDecimals(this.raw, otherStr, decimals), decimals);
  }

  div(other: string | number | Decimal, decimals = Math.max(this.defaultDecimals, 4)): Decimal {
    const otherStr = other instanceof Decimal ? other.raw : String(other);
    return new Decimal(divideDecimals(this.raw, otherStr, decimals), decimals);
  }

  abs(): Decimal {
    return new Decimal(this.raw.replace(/^-/, ''), this.defaultDecimals);
  }

  negate(): Decimal {
    if (this.isZero()) return new Decimal('0', this.defaultDecimals);
    return new Decimal(this.raw.startsWith('-') ? this.raw.slice(1) : `-${this.raw}`, this.defaultDecimals);
  }

  compareTo(other: string | number | Decimal): -1 | 0 | 1 {
    const otherNorm = normalizeDecimalInput(other instanceof Decimal ? other.raw : String(other));
    const selfNorm = normalizeDecimalInput(this.raw);
    const scale = Math.max(selfNorm.fracPart.length, otherNorm.fracPart.length);
    const bigA = toScaledBigInt(this.raw, scale);
    const bigB = toScaledBigInt(other, scale);
    if (bigA < bigB) return -1;
    if (bigA > bigB) return 1;
    return 0;
  }

  equals(other: string | number | Decimal): boolean {
    return this.compareTo(other) === 0;
  }

  gt(other: string | number | Decimal): boolean {
    return this.compareTo(other) === 1;
  }

  gte(other: string | number | Decimal): boolean {
    return this.compareTo(other) >= 0;
  }

  lt(other: string | number | Decimal): boolean {
    return this.compareTo(other) === -1;
  }

  lte(other: string | number | Decimal): boolean {
    return this.compareTo(other) <= 0;
  }

  isPositive(): boolean {
    return !this.isNegative() && !this.isZero();
  }

  isNegative(): boolean {
    return this.raw.startsWith('-') && !this.isZero();
  }

  isZero(): boolean {
    const norm = normalizeDecimalInput(this.raw);
    return norm.intPart === '0' && /^0*$/.test(norm.fracPart);
  }

  toRawString(): string {
    return this.raw;
  }

  toString(decimals = 2): string {
    return addDecimals(this.raw, '0', decimals);
  }

  static sum(values: Array<string | number | Decimal>, decimals = 2): Decimal {
    let acc = new Decimal('0', Math.max(decimals, 4));
    for (const v of values) {
      acc = acc.add(v, Math.max(decimals, 4));
    }
    return new Decimal(acc.toString(decimals), decimals);
  }
}

function getSeparators(locale?: SupportedLocale): { thousands: string; decimal: string } {
  const resolved = resolveIntlLocale(locale);
  return resolved === 'en-US'
    ? { thousands: ',', decimal: '.' }
    : { thousands: '.', decimal: ',' };
}

/**
 * Pure string-based number formatting preserving BigInt fixed-point precision.
 */
export function formatNumber(
  amount: string | number | Decimal | null | undefined,
  decimals = 2,
  locale: SupportedLocale = 'id-ID'
): string {
  const { thousands, decimal } = getSeparators(locale);
  if (amount === undefined || amount === null || amount === '') {
    return decimals > 0 ? `0${decimal}${'0'.repeat(decimals)}` : '0';
  }
  const raw = amount instanceof Decimal ? amount.toRawString() : String(amount).trim();
  if (!raw || raw === 'NaN' || raw === 'Infinity' || raw === '-Infinity') {
    return decimals > 0 ? `0${decimal}${'0'.repeat(decimals)}` : '0';
  }

  const normalized = addDecimals(raw, '0', decimals);
  const isNeg = normalized.startsWith('-');
  const clean = normalized.replace(/^-/, '');
  const [wholeRaw = '0', fracRaw = ''] = clean.split('.');
  const wholeFormatted = wholeRaw.replace(/\B(?=(\d{3})+(?!\d))/g, thousands);

  if (decimals <= 0) {
    return `${isNeg ? '-' : ''}${wholeFormatted}`;
  }

  return `${isNeg ? '-' : ''}${wholeFormatted}${decimal}${fracRaw}`;
}

/**
 * Enterprise currency formatter using Indonesian standard by default (dots for thousands, comma for decimals)
 * or English US standard (commas for thousands, dot for decimals).
 * Preserves full decimal string representation without lossy float casting.
 */
export function formatCurrency(
  amount: string | number | Decimal | null | undefined,
  currency = 'IDR',
  locale: SupportedLocale = 'id-ID',
  decimals = 2
): string {
  const { thousands, decimal } = getSeparators(locale);
  if (amount === undefined || amount === null || amount === '') {
    return decimals > 0 ? `0${decimal}${'0'.repeat(decimals)}` : '0';
  }
  const raw = amount instanceof Decimal ? amount.toRawString() : String(amount).trim();
  if (raw === 'NaN' || raw === '' || raw === 'Infinity' || raw === '-Infinity') {
    return decimals > 0 ? `0${decimal}${'0'.repeat(decimals)}` : '0';
  }

  const normalized = addDecimals(raw, '0', decimals);
  const isNegative = normalized.startsWith('-');
  const clean = normalized.replace(/^-/, '');
  const [wholeRaw = '0', fracRaw = ''] = clean.split('.');

  const wholeFormatted = wholeRaw.replace(/\B(?=(\d{3})+(?!\d))/g, thousands);
  const prefix = currency === 'IDR' ? 'Rp\u00a0' : `${currency}\u00a0`;

  if (decimals <= 0) {
    return `${isNegative ? '-' : ''}${prefix}${wholeFormatted}`;
  }

  return `${isNegative ? '-' : ''}${prefix}${wholeFormatted}${decimal}${fracRaw}`;
}

/**
 * Format exchange rate string with proper decimal precision.
 */
export function formatRate(
  rate: string | number | null | undefined,
  locale: SupportedLocale = 'id-ID'
): string {
  const { thousands, decimal } = getSeparators(locale);
  if (rate === undefined || rate === null || rate === '') {
    return `1${decimal}0000`;
  }
  const str = String(rate).trim();
  if (!str || str === 'NaN' || str === 'Infinity' || str === '-Infinity') {
    return `1${decimal}0000`;
  }
  const norm = normalizeDecimalInput(str);
  const isZero = norm.intPart === '0' && /^0*$/.test(norm.fracPart);
  const wholeFormatted = (norm.intPart.replace(/^0+(?=\d)/, '') || '0').replace(
    /\B(?=(\d{3})+(?!\d))/g,
    thousands
  );
  const sign = norm.isNeg && !isZero ? '-' : '';
  return norm.fracPart ? `${sign}${wholeFormatted}${decimal}${norm.fracPart}` : `${sign}${wholeFormatted}`;
}

export function formatDate(dateString: string, locale: SupportedLocale = 'id-ID'): string {
  if (!dateString) return '-';
  const d = new Date(dateString);
  if (Number.isNaN(d.getTime())) return dateString;
  return d.toLocaleDateString(resolveIntlLocale(locale), {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
  });
}

export function formatDateTime(dateString: string, locale: SupportedLocale = 'id-ID'): string {
  if (!dateString) return '-';
  const d = new Date(dateString);
  if (Number.isNaN(d.getTime())) return dateString;
  return d.toLocaleString(resolveIntlLocale(locale), {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}
