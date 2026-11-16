import { describe, expect, it } from 'vitest';
import {
  Decimal,
  addDecimals,
  divideDecimals,
  formatCurrency,
  formatRate,
  multiplyDecimals,
  subtractDecimals,
} from '../../src/lib/decimal';

describe('Fixed-point Decimal Engine', () => {
  it('handles addition, subtraction, multiplication, and division without float drift', () => {
    expect(addDecimals('0.1', '0.2', 2)).toBe('0.30');
    expect(subtractDecimals('1000.00', '0.01', 2)).toBe('999.99');
    expect(multiplyDecimals('19.99', '100', 2)).toBe('1999.00');
    expect(divideDecimals('100', '3', 4)).toBe('33.3333');
    expect(() => divideDecimals('10', '0', 2)).toThrow('Division by zero');
  });

  it('supports comparison helpers and summation on Decimal instances', () => {
    const a = new Decimal('1500.25');
    const b = new Decimal('1500.2500');
    const c = new Decimal('2500.00');

    expect(a.equals(b)).toBe(true);
    expect(c.gt(a)).toBe(true);
    expect(a.lt(c)).toBe(true);
    expect(Decimal.sum(['100.10', '200.20', '-50.05'], 2).toString(2)).toBe('250.25');
  });

  it('formats currency and FX rates in both id-ID and en-US locales', () => {
    expect(formatCurrency('1250000.5', 'IDR', 'id')).toBe('Rp\u00a01.250.000,50');
    expect(formatCurrency('1250000.5', 'IDR', 'en')).toBe('Rp\u00a01,250,000.50');
    expect(formatRate('16250.5000', 'id')).toBe('16.250,5000');
    expect(formatRate('16250.5000', 'en')).toBe('16,250.5000');
  });

  it('expands scientific notation inputs accurately without float corruption', () => {
    expect(addDecimals('1e-7', '0.0000002', 7)).toBe('0.0000003');
    expect(addDecimals('1.5e2', '25', 2)).toBe('175.00');
    expect(formatRate(1e-4, 'id')).toBe('0,0001');
    expect(new Decimal('1e-4').equals('0.0001')).toBe(true);
  });
});
