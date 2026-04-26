import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Trade, PnLSummary, SystemStats } from '../types';
import Card from '../components/Card';
import PnLValue from '../components/PnLValue';

export default function Dashboard() {
  const [activeTrades, setActiveTrades] = useState<Trade[]>([]);
  const [summary, setSummary] = useState<PnLSummary[]>([]);
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api.getActiveTrades(),
      api.getPnLCombined(),
      api.getSystemStats(),
    ]).then(([trades, sum, st]) => {
      setActiveTrades(trades);
      setSummary(sum);
      setStats(st);
    }).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="text-gray-500 py-8">Yukleniyor...</div>;

  const overall = summary.find(s => s.coin === 'ALL' && s.side === 'ALL');

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card
          title="Toplam Net K/Z"
          value={overall ? `${overall.total_net_pnl.toFixed(2)} USD` : '0 USD'}
          color={overall && overall.total_net_pnl > 0 ? 'green' : overall && overall.total_net_pnl < 0 ? 'red' : 'gray'}
        />
        <Card
          title="Aktif Pozisyon"
          value={activeTrades.length}
          color="blue"
        />
        <Card
          title="Islem Basari Orani"
          value={stats ? `%${stats.success_rate.toFixed(1)}` : '-'}
          color="green"
        />
        <Card
          title="Toplam Komisyon"
          value={overall ? `${overall.total_commission.toFixed(2)} USD` : '0 USD'}
          color="yellow"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
          <h2 className="text-lg font-semibold mb-4">Aktif Pozisyonlar</h2>
          {activeTrades.length === 0 ? (
            <p className="text-gray-500 text-sm">Aktif pozisyon yok</p>
          ) : (
            <div className="space-y-2">
              {activeTrades.slice(0, 10).map(t => (
                <div key={t.id} className="flex items-center justify-between bg-gray-800/50 rounded px-3 py-2 text-sm">
                  <div className="flex items-center gap-3">
                    <span className={`px-2 py-0.5 rounded text-xs font-bold ${
                      t.side === 'LONG' ? 'bg-emerald-900/50 text-emerald-400' : 'bg-red-900/50 text-red-400'
                    }`}>
                      {t.signal_type}
                    </span>
                    <span className="font-medium">{t.coin}</span>
                  </div>
                  <div className="text-gray-400 font-mono text-xs">
                    {t.quantity} @ {t.entry_price}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
          <h2 className="text-lg font-semibold mb-4">Coin Bazli Ozet</h2>
          {summary.filter(s => s.coin !== 'ALL' && s.side === 'ALL').length === 0 ? (
            <p className="text-gray-500 text-sm">Henuz islem yapilmamis</p>
          ) : (
            <div className="space-y-2">
              {summary.filter(s => s.coin !== 'ALL' && s.side === 'ALL').map(s => (
                <div key={s.coin} className="flex items-center justify-between bg-gray-800/50 rounded px-3 py-2 text-sm">
                  <span className="font-medium">{s.coin}</span>
                  <div className="flex items-center gap-4">
                    <span className="text-gray-500 text-xs">{s.trade_count} islem</span>
                    <PnLValue value={s.total_net_pnl} />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
