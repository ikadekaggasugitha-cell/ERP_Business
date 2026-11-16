import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export {
  Decimal,
  addDecimals,
  subtractDecimals,
  multiplyDecimals,
  divideDecimals,
  formatCurrency,
  formatNumber,
  formatRate,
  formatDate,
  formatDateTime,
  type SupportedLocale,
} from './decimal';
