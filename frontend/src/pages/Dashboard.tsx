import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Trade, PnLSummary, SystemStats, UnrealizedPnLResponse } from '../types';
import Card from '../components/Card';
import PnLValue from '../components/PnLValue';
import PnL24hOverview from '../components/PnL24hOverview';

export default function Dashboard() {
  const [activeTrades, setActiveTrades] = useState<Trade[]>([]);
  const [summary, setSummary] = useState<PnLSummary[]>([]);
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [unrealizedPnL, setUnrealizedPnL] = useState<UnrealizedPnLResponse | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let isMounted = true;

    const loadDashboard = async (showLoading = false) => {
      if (showLoading) setLoading(true);
      try {
        const [trades, sum, st, livePnL] = await Promise.all([
          api.getActiveTrades(),
          api.getPnLCombined(),
          api.getSystemStats(),
          api.getUnrealizedPnL(),
        ]);
        if (!isMounted) return;
        setActiveTrades(trades);
        setSummary(sum);
        setStats(st);
        setUnrealizedPnL(livePnL);
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    loadDashboard(true);
    const interval = window.setInterval(() => loadDashboard(), 60_000);

    return () => {
      isMounted = false;
      window.clearInterval(interval);
    };
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

      <PnL24hOverview compact />

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
        <div className="flex items-center justify-between gap-4 mb-4">
          <div>
            <h2 className="text-lg font-semibold">Anlik Kar/Zarar</h2>
            <p className="text-xs text-gray-500">
              Aktif pozisyonlar Binance guncel fiyatiyla 1 dakikada bir hesaplanir.
            </p>
          </div>
          <div className="text-right">
            <p className="text-xs text-gray-500">Toplam Net</p>
            <PnLValue value={unrealizedPnL?.total_net_pnl ?? 0} />
            {unrealizedPnL?.updated_at && (
              <p className="text-xs text-gray-600 mt-1">
                {new Date(unrealizedPnL.updated_at).toLocaleTimeString('tr-TR')}
              </p>
            )}
          </div>
        </div>

        {!unrealizedPnL || unrealizedPnL.items.length === 0 ? (
          <p className="text-gray-500 text-sm">Aktif pozisyon icin anlik K/Z yok</p>
        ) : (
          <div className="space-y-2">
            {unrealizedPnL.items.map(item => (
              <div key={item.trade_id} className="grid grid-cols-2 md:grid-cols-6 gap-3 bg-gray-800/50 rounded px-3 py-2 text-sm">
                <div>
                  <p className="text-xs text-gray-500">Coin</p>
                  <p className="font-medium">{item.coin} / {item.signal_type}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Yon</p>
                  <p className={item.side === 'LONG' ? 'text-emerald-400' : 'text-red-400'}>{item.side}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Giris</p>
                  <p className="font-mono">{item.entry_price}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Anlik Fiyat</p>
                  <p className="font-mono">{item.current_price}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Komisyon</p>
                  <p className="font-mono text-yellow-400">{item.total_commission.toFixed(4)}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Net K/Z</p>
                  <PnLValue value={item.net_pnl} />
                </div>
              </div>
            ))}
          </div>
        )}
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
