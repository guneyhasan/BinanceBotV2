import type { Config, Trade, PnLRecord, PnLSeries, PnLSummary, PnL24hResponse, UnrealizedPnLResponse, WebhookLog, SystemStats } from '../types';

const BASE = '/api';

async function fetchJSON<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(BASE + url, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
  return res.json();
}

export const api = {
  getConfig: () => fetchJSON<Config>('/config'),
  updateConfig: (data: Partial<Config>) =>
    fetchJSON<Config>('/config', { method: 'PUT', body: JSON.stringify(data) }),

  getTrades: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return fetchJSON<Trade[]>('/trades' + qs);
  },
  getActiveTrades: () => fetchJSON<Trade[]>('/trades/active'),

  getPnL: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return fetchJSON<PnLRecord[]>('/pnl' + qs);
  },
  getPnLSeries: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return fetchJSON<PnLSeries[]>('/pnl/series' + qs);
  },
  getPnLSummary: (coin?: string) => {
    const qs = coin ? '?coin=' + coin : '';
    return fetchJSON<PnLSummary[]>('/pnl/summary' + qs);
  },
  getPnLCombined: () => fetchJSON<PnLSummary[]>('/pnl/combined'),
  getPnLLast24h: () => fetchJSON<PnL24hResponse>('/pnl/last-24h'),
  getUnrealizedPnL: () => fetchJSON<UnrealizedPnLResponse>('/pnl/unrealized'),

  getWebhooks: (params?: Record<string, string>) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return fetchJSON<WebhookLog[]>('/webhooks' + qs);
  },
  getWebhookDetail: (id: number) => fetchJSON<WebhookLog>(`/webhooks/${id}`),

  getSystemStats: () => fetchJSON<SystemStats>('/system/stats'),
  testTelegram: (target: 'signal' | 'trade') =>
    fetchJSON<{ status: string; target: string }>('/telegram/test', {
      method: 'POST',
      body: JSON.stringify({ target }),
    }),
};
