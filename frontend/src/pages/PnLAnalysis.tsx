import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { PnLRecord, PnLSeries, PnLSummary } from '../types';
import PnLValue from '../components/PnLValue';
import Card from '../components/Card';

type Tab = 'summary' | 'records' | 'series';

export default function PnLAnalysis() {
  const [tab, setTab] = useState<Tab>('summary');
  const [summary, setSummary] = useState<PnLSummary[]>([]);
  const [records, setRecords] = useState<PnLRecord[]>([]);
  const [series, setSeries] = useState<PnLSeries[]>([]);
  const [coinFilter, setCoinFilter] = useState('');
  const [sideFilter, setSideFilter] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    const params: Record<string, string> = {};
    if (coinFilter) params.coin = coinFilter;
    if (sideFilter) params.side = sideFilter;

    if (tab === 'summary') {
      api.getPnLSummary(coinFilter).then(setSummary).finally(() => setLoading(false));
    } else if (tab === 'records') {
      api.getPnL(params).then(setRecords).finally(() => setLoading(false));
    } else {
      api.getPnLSeries(params).then(setSeries).finally(() => setLoading(false));
    }
  }, [tab, coinFilter, sideFilter]);

  const overall = summary.find(s => s.coin === 'ALL' && s.side === 'ALL');

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Kar/Zarar Analizi</h1>

      <div className="flex gap-3 flex-wrap items-center">
        <div className="flex bg-gray-800 rounded overflow-hidden">
          {(['summary', 'records', 'series'] as Tab[]).map(t => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`px-4 py-2 text-sm font-medium ${tab === t ? 'bg-emerald-600 text-white' : 'text-gray-400 hover:text-white'}`}
            >
              {t === 'summary' ? 'Ozet' : t === 'records' ? 'Detay' : 'Seri Bazli'}
            </button>
          ))}
        </div>
        <input
          placeholder="Coin..."
          className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
          value={coinFilter}
          onChange={e => setCoinFilter(e.target.value.toUpperCase())}
        />
        <select
          className="bg-gray-800 border border-gray-700 rounded px-3 py-1.5 text-sm"
          value={sideFilter}
          onChange={e => setSideFilter(e.target.value)}
        >
          <option value="">Tum Yonler</option>
          <option value="LONG">LONG (AL)</option>
          <option value="SHORT">SHORT (SAT)</option>
        </select>
      </div>

      {loading ? (
        <div className="text-gray-500 py-4">Yukleniyor...</div>
      ) : (
        <>
          {tab === 'summary' && (
            <div className="space-y-6">
              {overall && (
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <Card title="Toplam Brut K/Z" value={`${overall.total_gross_pnl.toFixed(2)} USD`} color={overall.total_gross_pnl >= 0 ? 'green' : 'red'} />
                  <Card title="Toplam Komisyon" value={`${overall.total_commission.toFixed(2)} USD`} color="yellow" />
                  <Card title="Toplam Net K/Z" value={`${overall.total_net_pnl.toFixed(2)} USD`} color={overall.total_net_pnl >= 0 ? 'green' : 'red'} />
                  <Card title="Kazanma Orani" value={`%${overall.win_rate.toFixed(1)}`} subtitle={`${overall.win_count}K / ${overall.loss_count}Z`} color={overall.win_rate >= 50 ? 'green' : 'red'} />
                </div>
              )}

              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-left text-gray-500 border-b border-gray-800">
                      <th className="pb-2 pr-4">Coin</th>
                      <th className="pb-2 pr-4">Yon</th>
                      <th className="pb-2 pr-4">Islem</th>
                      <th className="pb-2 pr-4">Brut K/Z</th>
                      <th className="pb-2 pr-4">Komisyon</th>
                      <th className="pb-2 pr-4">Net K/Z</th>
                      <th className="pb-2">Kazanma %</th>
                    </tr>
                  </thead>
                  <tbody>
                    {summary.filter(s => !(s.coin === 'ALL' && s.side === 'ALL')).map((s, i) => (
                      <tr key={i} className={`border-b border-gray-800/50 ${s.side === 'ALL' ? 'bg-gray-800/20 font-semibold' : ''}`}>
                        <td className="py-2 pr-4">{s.coin}</td>
                        <td className="py-2 pr-4 text-gray-400">{s.side}</td>
                        <td className="py-2 pr-4">{s.trade_count}</td>
                        <td className="py-2 pr-4"><PnLValue value={s.total_gross_pnl} /></td>
                        <td className="py-2 pr-4 text-yellow-400 font-mono">{s.total_commission.toFixed(4)}</td>
                        <td className="py-2 pr-4"><PnLValue value={s.total_net_pnl} /></td>
                        <td className="py-2">{s.win_rate.toFixed(1)}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {tab === 'records' && (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-gray-500 border-b border-gray-800">
                    <th className="pb-2 pr-4">Zaman</th>
                    <th className="pb-2 pr-4">Coin</th>
                    <th className="pb-2 pr-4">Kapatilan</th>
                    <th className="pb-2 pr-4">Kapatan</th>
                    <th className="pb-2 pr-4">Giris</th>
                    <th className="pb-2 pr-4">Cikis</th>
                    <th className="pb-2 pr-4">Miktar</th>
                    <th className="pb-2 pr-4">Brut</th>
                    <th className="pb-2 pr-4">Komisyon</th>
                    <th className="pb-2">Net</th>
                  </tr>
                </thead>
                <tbody>
                  {records.map(r => (
                    <tr key={r.id} className="border-b border-gray-800/50 hover:bg-gray-800/30">
                      <td className="py-2 pr-4 text-gray-400 font-mono text-xs">{new Date(r.closed_at).toLocaleString('tr-TR')}</td>
                      <td className="py-2 pr-4 font-medium">{r.coin}</td>
                      <td className="py-2 pr-4">
                        <span className={`px-1.5 py-0.5 rounded text-xs font-bold ${
                          r.closed_signal_type.startsWith('AL') ? 'bg-emerald-900/50 text-emerald-400' : 'bg-red-900/50 text-red-400'
                        }`}>{r.closed_signal_type}</span>
                      </td>
                      <td className="py-2 pr-4">
                        <span className={`px-1.5 py-0.5 rounded text-xs font-bold ${
                          r.closing_signal_type.startsWith('AL') ? 'bg-emerald-900/50 text-emerald-400' : 'bg-red-900/50 text-red-400'
                        }`}>{r.closing_signal_type}</span>
                      </td>
                      <td className="py-2 pr-4 font-mono text-xs">{r.entry_price}</td>
                      <td className="py-2 pr-4 font-mono text-xs">{r.exit_price}</td>
                      <td className="py-2 pr-4 font-mono text-xs">{r.quantity}</td>
                      <td className="py-2 pr-4"><PnLValue value={r.gross_pnl} /></td>
                      <td className="py-2 pr-4 text-yellow-400 font-mono text-xs">{r.total_commission.toFixed(4)}</td>
                      <td className="py-2"><PnLValue value={r.net_pnl} /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {tab === 'series' && (
            <div className="space-y-4">
              {series.length === 0 ? (
                <p className="text-gray-500 text-sm">Seri verisi bulunamadi</p>
              ) : (
                series.map(s => (
                  <div key={s.series_index} className="bg-gray-900 border border-gray-800 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <span className="text-sm font-bold text-gray-300">Seri #{s.series_index}</span>
                        <span className="text-sm text-gray-500">{s.coin}</span>
                        <div className="flex gap-1">
                          {s.sequence.map((sig, i) => (
                            <span key={i} className={`px-1.5 py-0.5 rounded text-xs font-bold ${
                              sig.startsWith('AL') ? 'bg-emerald-900/50 text-emerald-400' : 'bg-red-900/50 text-red-400'
                            }`}>{sig}</span>
                          ))}
                        </div>
                      </div>
                      <div className="flex items-center gap-4 text-sm">
                        <span className="text-gray-500">Brut: <PnLValue value={s.total_gross_pnl} /></span>
                        <span className="text-gray-500">Kom: <span className="text-yellow-400 font-mono">{s.total_commission.toFixed(4)}</span></span>
                        <span className="text-gray-500">Net: <PnLValue value={s.total_net_pnl} /></span>
                      </div>
                    </div>
                    <div className="space-y-1">
                      {s.records.map(r => (
                        <div key={r.id} className="flex items-center justify-between bg-gray-800/30 rounded px-3 py-1.5 text-xs">
                          <div className="flex items-center gap-3">
                            <span className="text-gray-500">{new Date(r.closed_at).toLocaleString('tr-TR')}</span>
                            <span>{r.closed_signal_type} &rarr; {r.closing_signal_type}</span>
                          </div>
                          <PnLValue value={r.net_pnl} />
                        </div>
                      ))}
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
