import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Trade } from '../types';

export default function Trades() {
  const [trades, setTrades] = useState<Trade[]>([]);
  const [filters, setFilters] = useState({ coin: '', signal_type: '', active: '' });
  const [loading, setLoading] = useState(true);

  const load = () => {
    setLoading(true);
    const params: Record<string, string> = {};
    if (filters.coin) params.coin = filters.coin;
    if (filters.signal_type) params.signal_type = filters.signal_type;
    if (filters.active) params.active = filters.active;
    api.getTrades(params).then(setTrades).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [filters]);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Islemler</h1>

      <div className="flex gap-3 flex-wrap">
        <input
          placeholder="Coin filtresi..."
          className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
          value={filters.coin}
          onChange={e => setFilters(f => ({ ...f, coin: e.target.value.toUpperCase() }))}
        />
        <select
          className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
          value={filters.signal_type}
          onChange={e => setFilters(f => ({ ...f, signal_type: e.target.value }))}
        >
          <option value="">Tum Sinyaller</option>
          {['AL1','AL2','AL3','SAT1','SAT2','SAT3'].map(s => <option key={s} value={s}>{s}</option>)}
        </select>
        <select
          className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
          value={filters.active}
          onChange={e => setFilters(f => ({ ...f, active: e.target.value }))}
        >
          <option value="">Tumu</option>
          <option value="true">Aktif</option>
          <option value="false">Pasif</option>
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
                <th className="pb-2 pr-4">ID</th>
                <th className="pb-2 pr-4">Zaman</th>
                <th className="pb-2 pr-4">Coin</th>
                <th className="pb-2 pr-4">Sinyal</th>
                <th className="pb-2 pr-4">Yon</th>
                <th className="pb-2 pr-4">Hesap</th>
                <th className="pb-2 pr-4">Miktar</th>
                <th className="pb-2 pr-4">Fiyat</th>
                <th className="pb-2 pr-4">Kaldırac</th>
                <th className="pb-2 pr-4">Komisyon</th>
                <th className="pb-2">Durum</th>
              </tr>
            </thead>
            <tbody>
              {trades.map(t => (
                <tr key={t.id} className="border-b border-gray-800/50 hover:bg-gray-800/30">
                  <td className="py-2 pr-4 text-gray-500">{t.id}</td>
                  <td className="py-2 pr-4 text-gray-400 font-mono text-xs">{new Date(t.created_at).toLocaleString('tr-TR')}</td>
                  <td className="py-2 pr-4 font-medium">{t.coin}</td>
                  <td className="py-2 pr-4">
                    <span className={`px-2 py-0.5 rounded text-xs font-bold ${
                      t.signal_type.startsWith('AL') ? 'bg-emerald-900/50 text-emerald-400' : 'bg-red-900/50 text-red-400'
                    }`}>
                      {t.signal_type}
                    </span>
                  </td>
                  <td className="py-2 pr-4">
                    <span className={t.side === 'LONG' ? 'text-emerald-400' : 'text-red-400'}>{t.side}</span>
                  </td>
                  <td className="py-2 pr-4 text-gray-400 text-xs">{t.account_type}</td>
                  <td className="py-2 pr-4 font-mono">{t.quantity}</td>
                  <td className="py-2 pr-4 font-mono">{t.entry_price}</td>
                  <td className="py-2 pr-4">{t.leverage}x</td>
                  <td className="py-2 pr-4 font-mono text-yellow-400">{t.commission.toFixed(4)}</td>
                  <td className="py-2">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      t.is_active ? 'bg-emerald-900/50 text-emerald-400' : 'bg-gray-800 text-gray-500'
                    }`}>
                      {t.is_active ? 'Aktif' : 'Pasif'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
