import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { WebhookLog } from '../types';
import StatusBadge from '../components/StatusBadge';

export default function Webhooks() {
  const [webhooks, setWebhooks] = useState<WebhookLog[]>([]);
  const [selected, setSelected] = useState<WebhookLog | null>(null);
  const [filters, setFilters] = useState({ coin: '', signal: '', status: '' });
  const [loading, setLoading] = useState(true);

  const load = () => {
    setLoading(true);
    const params: Record<string, string> = {};
    if (filters.coin) params.coin = filters.coin;
    if (filters.signal) params.signal = filters.signal;
    if (filters.status) params.status = filters.status;
    api.getWebhooks(params).then(setWebhooks).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [filters]);

  const openDetail = async (id: number) => {
    const detail = await api.getWebhookDetail(id);
    setSelected(detail);
  };

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Gelen Istekler</h1>

      <div className="flex gap-3 flex-wrap">
        <input
          placeholder="Coin filtresi..."
          className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
          value={filters.coin}
          onChange={e => setFilters(f => ({ ...f, coin: e.target.value.toUpperCase() }))}
        />
        <select
          className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
          value={filters.signal}
          onChange={e => setFilters(f => ({ ...f, signal: e.target.value }))}
        >
          <option value="">Tum Sinyaller</option>
          {['AL1','AL2','AL3','SAT1','SAT2','SAT3'].map(s => <option key={s} value={s}>{s}</option>)}
        </select>
        <select
          className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
          value={filters.status}
          onChange={e => setFilters(f => ({ ...f, status: e.target.value }))}
        >
          <option value="">Tum Durumlar</option>
          {['RECEIVED','PROCESSING','COMPLETED','RETRY_SUCCESS','FAILED'].map(s => <option key={s} value={s}>{s}</option>)}
        </select>
        <button onClick={load} className="bg-gray-700 hover:bg-gray-600 px-3 py-1.5 rounded text-sm">Yenile</button>
      </div>

      {loading ? (
        <div className="text-gray-500 py-4">Yukleniyor...</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 border-b border-gray-800">
                <th className="pb-2 pr-4">Zaman</th>
                <th className="pb-2 pr-4">Coin</th>
                <th className="pb-2 pr-4">Sinyal</th>
                <th className="pb-2 pr-4">Durum</th>
                <th className="pb-2 pr-4">Retry</th>
                <th className="pb-2">Detay</th>
              </tr>
            </thead>
            <tbody>
              {webhooks.map(w => (
                <tr key={w.id} className="border-b border-gray-800/50 hover:bg-gray-800/30">
                  <td className="py-2 pr-4 text-gray-400 font-mono text-xs">{new Date(w.received_at).toLocaleString('tr-TR')}</td>
                  <td className="py-2 pr-4 font-medium">{w.coin}</td>
                  <td className="py-2 pr-4">
                    <span className={`px-2 py-0.5 rounded text-xs font-bold ${
                      w.signal_type.startsWith('AL') ? 'bg-emerald-900/50 text-emerald-400' : 'bg-red-900/50 text-red-400'
                    }`}>
                      {w.signal_type}
                    </span>
                  </td>
                  <td className="py-2 pr-4"><StatusBadge status={w.status} /></td>
                  <td className="py-2 pr-4 text-gray-400">{w.retry_count}</td>
                  <td className="py-2">
                    <button
                      onClick={() => openDetail(w.id)}
                      className="text-blue-400 hover:text-blue-300 text-xs"
                    >
                      Goruntule
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {selected && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4" onClick={() => setSelected(null)}>
          <div className="bg-gray-900 border border-gray-700 rounded-lg max-w-3xl w-full max-h-[80vh] overflow-y-auto p-6" onClick={e => e.stopPropagation()}>
            <div className="flex justify-between items-start mb-4">
              <h2 className="text-lg font-bold">Webhook Detayi</h2>
              <button onClick={() => setSelected(null)} className="text-gray-500 hover:text-gray-300">&times;</button>
            </div>

            <div className="grid grid-cols-2 gap-4 text-sm mb-4">
              <div><span className="text-gray-500">Request ID:</span> <span className="font-mono text-xs">{selected.request_id}</span></div>
              <div><span className="text-gray-500">Durum:</span> <StatusBadge status={selected.status} /></div>
              <div><span className="text-gray-500">Coin:</span> {selected.coin}</div>
              <div><span className="text-gray-500">Sinyal:</span> {selected.signal_type}</div>
              <div><span className="text-gray-500">Retry:</span> {selected.retry_count}</div>
              <div><span className="text-gray-500">Trade ID:</span> {selected.trade_id ?? '-'}</div>
            </div>

            {selected.error_message && (
              <div className="bg-red-900/20 border border-red-800 rounded p-3 mb-4 text-sm text-red-400">
                {selected.error_message}
              </div>
            )}

            <div className="mb-3">
              <h3 className="text-sm font-semibold text-gray-400 mb-2">Ham Request</h3>
              <pre className="bg-gray-800 rounded p-3 text-xs overflow-x-auto">{JSON.stringify(selected.raw_body, null, 2)}</pre>
            </div>

            {selected.executions && selected.executions.length > 0 && (
              <div>
                <h3 className="text-sm font-semibold text-gray-400 mb-2">Islem Adimlari</h3>
                <div className="space-y-2">
                  {selected.executions.map(e => (
                    <div key={e.id} className="bg-gray-800/50 rounded p-3 text-xs">
                      <div className="flex items-center gap-3 mb-1">
                        <span className="text-gray-500">Deneme #{e.attempt_number}</span>
                        <span className="font-medium">{e.step}</span>
                        <StatusBadge status={e.status} />
                        {e.account_type && <span className="text-gray-500">{e.account_type}</span>}
                        {e.duration_ms != null && <span className="text-gray-600">{e.duration_ms}ms</span>}
                      </div>
                      {e.error_message && <div className="text-red-400 mt-1">{String(e.error_message)}</div>}
                      {e.binance_response != null && (
                        <details className="mt-1">
                          <summary className="text-gray-600 cursor-pointer">Binance Yaniti</summary>
                          <pre className="mt-1 text-gray-500 overflow-x-auto">{JSON.stringify(e.binance_response, null, 2)}</pre>
                        </details>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
