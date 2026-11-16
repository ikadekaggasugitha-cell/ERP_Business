import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { apiClient, resolveApiBaseUrl } from './api';
import { emitErpEvent } from './event-bus';
import i18n from './i18n';
import {
  extractEventEnvelope,
  KNOWN_SSE_EVENT_TYPES,
  resolveInvalidatedQueryKeys,
  type NormalizedSseEvent,
} from './realtime-matcher';
import { useAuthStore } from '../store/auth-store';
import { useUiStore } from '../store/ui-store';

export { extractEventEnvelope, KNOWN_SSE_EVENT_TYPES, resolveInvalidatedQueryKeys };
export type { NormalizedSseEvent };

export const REALTIME_QUERY_KEYS = [
  ['journals'],
  ['balances'],
  ['gl-balances'],
  ['coa-accounts'],
  ['fiscal-periods'],
  ['approvals'],
  ['ap-invoices'],
  ['ar-invoices'],
  ['tax-invoices'],
  ['bank-accounts'],
  ['cashbank-transactions'],
  ['inventory-items'],
  ['purchase-orders'],
  ['sales-orders'],
  ['work-orders'],
  ['employees'],
  ['payroll-runs'],
  ['currencies'],
  ['exchange-rates'],
  ['business-partners'],
  ['organizations'],
  ['users'],
  ['roles'],
  ['audit-events'],
] as const;

const MAX_RECONNECT_DELAY_MS = 30_000;
const BASE_RECONNECT_DELAY_MS = 1_500;

export function useRealtimeEvents() {
  const queryClient = useQueryClient();
  const { isAuthenticated, isDemoSession, token, activeOrg } = useAuthStore();
  const { demoMode, addNotification, setRealtimeConnected } = useUiStore();

  useEffect(() => {
    const isOfflineDemo =
      isDemoSession || demoMode || token === 'dev-jwt-offline-token';

    if (!isAuthenticated || !activeOrg || isOfflineDemo) {
      setRealtimeConnected(false);
      return;
    }

    const currentOrgId = activeOrg.id;
    let eventSource: EventSource | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let reconnectAttempt = 0;
    let isDisposed = false;

    const handleEvent = (event: MessageEvent) => {
      if (isDisposed) return;

      const { eventKey, payload } = extractEventEnvelope(event.data, event.type);

      const queryKeys = resolveInvalidatedQueryKeys(eventKey);
      for (const qk of queryKeys) {
        void queryClient.invalidateQueries({ queryKey: qk });
      }

      emitErpEvent('erp:realtime-event', { eventKey, payload });

      if (/^(?:erp\.)?gl\.journal\.posted/.test(eventKey)) {
        const journalNo = String(payload.journalNo || 'JV-NEW');
        const title = i18n.t('realtime.journalPostedTitle');
        const message = i18n.t('realtime.journalPostedDesc', { journalNo });
        addNotification({ title, message, type: 'success' });
        toast.success(title, { description: message });
      } else if (/^(?:erp\.)?workflow\.approval\.completed/.test(eventKey)) {
        const documentNo = String(
          payload.documentNo || payload.documentRef || payload.documentId || 'DOC'
        );
        const approver = String(
          payload.approverName || payload.approver || i18n.t('realtime.defaultApprover')
        );
        const title = i18n.t('realtime.approvalCompletedTitle');
        const message = i18n.t('realtime.approvalCompletedDesc', { documentNo, approver });
        addNotification({ title, message, type: 'info' });
        toast.info(title, { description: message });
      } else if (/^(?:erp\.)?inventory\.reservation\.failed/.test(eventKey)) {
        const itemCode = String(payload.itemCode || payload.itemId || 'SKU');
        const title = i18n.t('realtime.stockFailedTitle');
        const message = i18n.t('realtime.stockFailedDesc', { itemCode });
        addNotification({ title, message, type: 'warning' });
        toast.warning(title, { description: message });
      }
    };

    const scheduleReconnect = () => {
      if (isDisposed || reconnectTimer !== null) return;
      const delay = Math.min(
        MAX_RECONNECT_DELAY_MS,
        BASE_RECONNECT_DELAY_MS * Math.pow(2, reconnectAttempt)
      );
      reconnectAttempt = Math.min(reconnectAttempt + 1, 5);
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        if (!isDisposed) {
          void connect();
        }
      }, delay);
    };

    async function connect() {
      if (isDisposed) return;

      try {
        let ticketQuery = '';
        try {
          const ticketRes = await apiClient<{ ticket: string }>('/auth/stream-ticket', {
            method: 'POST',
            skipAuthRedirect: true,
          });
          if (ticketRes?.ticket) {
            ticketQuery = `ticket=${encodeURIComponent(ticketRes.ticket)}&`;
          }
        } catch {
          // Fallback to HttpOnly session cookie (withCredentials: true)
        }

        if (isDisposed) return;

        const base = resolveApiBaseUrl();
        const sseUrl = `${base}/stream/events?${ticketQuery}orgId=${encodeURIComponent(currentOrgId)}`;
        eventSource = new EventSource(sseUrl, { withCredentials: true });

        eventSource.onopen = () => {
          if (!isDisposed) {
            reconnectAttempt = 0;
            setRealtimeConnected(true);
          }
        };

        eventSource.onmessage = handleEvent;

        for (const evtType of KNOWN_SSE_EVENT_TYPES) {
          eventSource.addEventListener(evtType, handleEvent as EventListener);
        }

        eventSource.onerror = () => {
          if (isDisposed) return;
          setRealtimeConnected(false);
          if (eventSource) {
            eventSource.close();
            eventSource = null;
          }
          scheduleReconnect();
        };
      } catch {
        if (!isDisposed) {
          setRealtimeConnected(false);
          scheduleReconnect();
        }
      }
    }

    void connect();

    return () => {
      isDisposed = true;
      setRealtimeConnected(false);
      if (reconnectTimer !== null) {
        clearTimeout(reconnectTimer);
        reconnectTimer = null;
      }
      if (eventSource) {
        eventSource.close();
        eventSource = null;
      }
    };
  }, [
    isAuthenticated,
    isDemoSession,
    demoMode,
    token,
    activeOrg,
    queryClient,
    addNotification,
    setRealtimeConnected,
  ]);
}
