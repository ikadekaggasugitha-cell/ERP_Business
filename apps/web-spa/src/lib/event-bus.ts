import type { AppLocale, AppTheme, Organization, PortalMode } from '../types';

export interface ErpEventMap {
  'erp:org-changed': { org: Organization | null };
  'erp:locale-changed': { locale: AppLocale };
  'erp:theme-changed': { theme: AppTheme };
  'erp:portal-changed': { portal: PortalMode };
  'erp:realtime-event': { eventKey: string; payload: Record<string, unknown> };
  'erp:unauthorized': { status: number; correlationId?: string };
  'erp:conflict': { status: number; message: string; correlationId?: string };
  'erp:navigate': { to: string };
}

export type ErpEventName = keyof ErpEventMap;

const fallbackTarget = new EventTarget();

function getBusTarget(): EventTarget {
  if (typeof window !== 'undefined') {
    return window;
  }
  return fallbackTarget;
}

export function emitErpEvent<K extends ErpEventName>(type: K, detail: ErpEventMap[K]): void {
  const target = getBusTarget();
  const event = new CustomEvent<ErpEventMap[K]>(type, { detail });
  target.dispatchEvent(event);
}

export function onErpEvent<K extends ErpEventName>(
  type: K,
  handler: (detail: ErpEventMap[K]) => void
): () => void {
  const target = getBusTarget();
  const listener = (evt: Event) => {
    const custom = evt as CustomEvent<ErpEventMap[K]>;
    handler(custom.detail);
  };
  target.addEventListener(type, listener);
  return () => {
    target.removeEventListener(type, listener);
  };
}
