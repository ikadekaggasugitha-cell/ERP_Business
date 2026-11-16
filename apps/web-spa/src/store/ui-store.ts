import { create } from 'zustand';
import i18n from '../lib/i18n';
import { emitErpEvent } from '../lib/event-bus';
import type { AppLocale, AppTheme, PortalMode } from '../types';

export interface UiNotification {
  id: string;
  title: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error';
  timestamp: string;
  read: boolean;
}

function readStorage<T extends string>(key: string, allowed: readonly T[], fallback: T): T {
  if (typeof window === 'undefined') return fallback;
  try {
    const val = window.localStorage.getItem(key);
    if (val && (allowed as readonly string[]).includes(val)) {
      return val as T;
    }
  } catch {
    // Ignore storage read restrictions
  }
  return fallback;
}

function writeStorage(key: string, value: string): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(key, value);
  } catch {
    // Ignore storage write restrictions
  }
}

export function applyThemeToDom(theme: AppTheme): void {
  if (typeof document === 'undefined') return;
  const root = document.documentElement;
  root.classList.toggle('dark', theme === 'dark');
  root.style.colorScheme = theme;
}

export function applyLocaleToDom(locale: AppLocale): void {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale;
  }
  if (i18n.language !== locale) {
    void i18n.changeLanguage(locale);
  }
}

interface UiState {
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  mobileDrawerOpen: boolean;
  setMobileDrawerOpen: (open: boolean) => void;
  toggleMobileDrawer: () => void;
  commandPaletteOpen: boolean;
  setCommandPaletteOpen: (open: boolean) => void;
  theme: AppTheme;
  setTheme: (theme: AppTheme) => void;
  toggleTheme: () => void;
  locale: AppLocale;
  setLocale: (locale: AppLocale) => void;
  portal: PortalMode;
  setPortal: (portal: PortalMode) => void;
  demoMode: boolean;
  setDemoMode: (demo: boolean) => void;
  isRealtimeConnected: boolean;
  setRealtimeConnected: (connected: boolean) => void;
  notifications: UiNotification[];
  addNotification: (notification: Omit<UiNotification, 'id' | 'timestamp' | 'read'>) => void;
  markAsRead: (id: string) => void;
  markAllAsRead: () => void;
  clearNotifications: () => void;
}

const initialTheme = readStorage<AppTheme>('erp_theme', ['light', 'dark'], 'light');
const initialLocale = readStorage<AppLocale>('erp_locale', ['id', 'en'], 'id');
const initialPortal = readStorage<PortalMode>('erp_portal', ['backoffice', 'admin', 'reporting'], 'backoffice');

applyThemeToDom(initialTheme);
applyLocaleToDom(initialLocale);

export const useUiStore = create<UiState>((set, get) => ({
  sidebarCollapsed: false,
  toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
  setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),

  mobileDrawerOpen: false,
  setMobileDrawerOpen: (open) => set({ mobileDrawerOpen: open }),
  toggleMobileDrawer: () => set((state) => ({ mobileDrawerOpen: !state.mobileDrawerOpen })),

  commandPaletteOpen: false,
  setCommandPaletteOpen: (open) => set({ commandPaletteOpen: open }),

  theme: initialTheme,
  setTheme: (theme) => {
    writeStorage('erp_theme', theme);
    applyThemeToDom(theme);
    emitErpEvent('erp:theme-changed', { theme });
    set({ theme });
  },
  toggleTheme: () => {
    const next: AppTheme = get().theme === 'dark' ? 'light' : 'dark';
    get().setTheme(next);
  },

  locale: initialLocale,
  setLocale: (locale) => {
    writeStorage('erp_locale', locale);
    applyLocaleToDom(locale);
    emitErpEvent('erp:locale-changed', { locale });
    set({ locale });
  },

  portal: initialPortal,
  setPortal: (portal) => {
    writeStorage('erp_portal', portal);
    emitErpEvent('erp:portal-changed', { portal });
    set({ portal });
  },

  demoMode: false,
  setDemoMode: (demoMode) => set({ demoMode }),

  isRealtimeConnected: false,
  setRealtimeConnected: (connected) => set({ isRealtimeConnected: connected }),

  notifications: [],

  addNotification: (n) =>
    set((state) => ({
      notifications: [
        {
          ...n,
          id: `notif-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
          timestamp: new Date().toISOString(),
          read: false,
        },
        ...state.notifications,
      ].slice(0, 50),
    })),

  markAsRead: (id) =>
    set((state) => ({
      notifications: state.notifications.map((n) => (n.id === id ? { ...n, read: true } : n)),
    })),

  markAllAsRead: () =>
    set((state) => ({
      notifications: state.notifications.map((n) => (n.read ? n : { ...n, read: true })),
    })),

  clearNotifications: () => set({ notifications: [] }),
}));
