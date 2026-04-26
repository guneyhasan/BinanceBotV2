import { useEffect, useMemo, useState } from 'react';
import { api } from '../api/client';
import type { PnL24hResponse } from '../types';
import Card from './Card';
import PnLValue from './PnLValue';

interface PnL24hOverviewProps {
  compact?: boolean;
}

function pnlColor(value: number): 'green' | 'red' | 'gray' {
  if (value > 0) return 'green';
  if (value < 0) return 'red';
  return 'gray';
}

function formatUsd(value: number) {
  return `${value.toFixed(2)} USD`;
}

function accountLabel(accountType: string) {
  if (accountType === 'AL_ACCOUNT') return 'AL Hesabi';
  if (accountType === 'SAT_ACCOUNT') return 'SAT Hesabi';
  return accountType;
}

function PnLChart({ data }: { data: PnL24hResponse['chart'] }) {
  const points = useMemo(() => {
    if (data.length === 0) return '';

    const values = data.map(point => point.cumulative_realized_pnl);
    const min = Math.min(0, ...values);
    const max = Math.max(0, ...values);
    const range = max - min || 1;
    const width = 320;
    const height = 120;
    const pad = 12;

    return data.map((point, index) => {
      const x = data.length === 1 ? width / 2 : pad + (index / (data.length - 1)) * (width - pad * 2);
      const y = height - pad - ((point.cumulative_realized_pnl - min) / range) * (height - pad * 2);
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    }).join(' ');
  }, [data]);

  const lastPoint = data[data.length - 1];

  return (
    <div className="rounded-lg border border-gray-800 bg-gray-950/60 p-4">
      <div className="flex items-center justify-between gap-3 mb-3">
        <div>
          <h3 className="text-sm font-semibold text-gray-200">Son 24 Saat Grafik</h3>
          <p className="text-xs text-gray-500">Saatlik gerçekleşmiş kümülatif net K/Z</p>
        </div>
        <div className="text-right text-xs text-gray-500">
          <p>Guncel</p>
          <PnLValue value={lastPoint?.cumulative_realized_pnl ?? 0} />
        </div>
      </div>
      <svg viewBox="0 0 320 120" className="h-40 w-full overflow-visible">
        <line x1="12" y1="108" x2="308" y2="108" className="stroke-gray-800" strokeWidth="1" />
        <line x1="12" y1="60" x2="308" y2="60" className="stroke-gray-800/70" strokeWidth="1" strokeDasharray="4 4" />
        {points ? (
          <polyline
            fill="none"
            points={points}
            className={(lastPoint?.cumulative_realized_pnl ?? 0) >= 0 ? 'stroke-emerald-400' : 'stroke-red-400'}
            strokeWidth="3"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        ) : (
          <text x="160" y="64" textAnchor="middle" className="fill-gray-500 text-xs">
            Veri yok
          </text>
        )}
      </svg>
    </div>
  );
}

export default function PnL24hOverview({ compact = false }: PnL24hOverviewProps) {
  const [data, setData] = useState<PnL24hResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    const load = async (showLoading = false) => {
      if (showLoading) setLoading(true);
      try {
        const response = await api.getPnLLast24h();
        if (!isMounted) return;
        setData(response);
        setError(null);
      } catch (err) {
        if (!isMounted) return;
        setError(err instanceof Error ? err.message : '24s K/Z verisi alinamadi');
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    load(true);
    const interval = window.setInterval(() => load(), 60_000);

    return () => {
      isMounted = false;
      window.clearInterval(interval);
    };
  }, []);

  const alAccount = data?.accounts.find(account => account.account_type === 'AL_ACCOUNT');
  const satAccount = data?.accounts.find(account => account.account_type === 'SAT_ACCOUNT');
  const visibleCoins = compact ? data?.coins.slice(0, 5) ?? [] : data?.coins ?? [];

  if (loading) {
    return <div className="text-gray-500 py-4">Son 24 saat K/Z yukleniyor...</div>;
  }

  if (error) {
    return (
      <div className="bg-red-950/30 border border-red-900/50 rounded-lg p-4 text-sm text-red-300">
        {error}
      </div>
    );
  }

  if (!data) return null;

  return (
    <section className="space-y-4">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold">Son 24 Saat K/Z</h2>
          <p className="text-xs text-gray-500">
            {new Date(data.from).toLocaleString('tr-TR')} - {new Date(data.to).toLocaleString('tr-TR')}
          </p>
        </div>
        <p className="text-xs text-gray-500">
          Guncelleme: {new Date(data.updated_at).toLocaleTimeString('tr-TR')}
        </p>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Card
          title="AL Hesabi Net"
          value={formatUsd(alAccount?.total_net_pnl ?? 0)}
          subtitle={`Realized ${formatUsd(alAccount?.realized_net_pnl ?? 0)} / Anlik ${formatUsd(alAccount?.unrealized_net_pnl ?? 0)}`}
          color={pnlColor(alAccount?.total_net_pnl ?? 0)}
        />
        <Card
          title="SAT Hesabi Net"
          value={formatUsd(satAccount?.total_net_pnl ?? 0)}
          subtitle={`Realized ${formatUsd(satAccount?.realized_net_pnl ?? 0)} / Anlik ${formatUsd(satAccount?.unrealized_net_pnl ?? 0)}`}
          color={pnlColor(satAccount?.total_net_pnl ?? 0)}
        />
        <Card
          title="Toplam Net"
          value={formatUsd(data.total_net_pnl)}
          subtitle={`Realized ${formatUsd(data.total_realized_net_pnl)} / Anlik ${formatUsd(data.total_unrealized_net_pnl)}`}
          color={pnlColor(data.total_net_pnl)}
        />
      </div>

      <div className={compact ? 'grid grid-cols-1 gap-4' : 'grid grid-cols-1 lg:grid-cols-5 gap-4'}>
        <div className={compact ? '' : 'lg:col-span-3'}>
          <PnLChart data={data.chart} />
        </div>

        <div className={`bg-gray-900 border border-gray-800 rounded-lg p-4 ${compact ? '' : 'lg:col-span-2'}`}>
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-gray-200">Coin Bazli</h3>
            {compact && data.coins.length > visibleCoins.length && (
              <span className="text-xs text-gray-500">Ilk {visibleCoins.length}</span>
            )}
          </div>

          {visibleCoins.length === 0 ? (
            <p className="text-gray-500 text-sm">Son 24 saatte K/Z verisi yok</p>
          ) : (
            <div className="space-y-2">
              {visibleCoins.map(coin => (
                <div key={coin.coin} className="rounded bg-gray-800/50 px-3 py-2 text-sm">
                  <div className="flex items-center justify-between gap-3">
                    <span className="font-medium">{coin.coin}</span>
                    <PnLValue value={coin.total_net_pnl} />
                  </div>
                  <div className="mt-1 grid grid-cols-2 gap-2 text-xs text-gray-500">
                    <span>AL: <PnLValue value={coin.al_net_pnl} /></span>
                    <span>SAT: <PnLValue value={coin.sat_net_pnl} /></span>
                    <span>Realized: <PnLValue value={coin.realized_net_pnl} /></span>
                    <span>Anlik: <PnLValue value={coin.unrealized_net_pnl} /></span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {!compact && (
        <div className="overflow-x-auto rounded-lg border border-gray-800">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500 bg-gray-900">
                <th className="px-3 py-2">Hesap</th>
                <th className="px-3 py-2">Islem</th>
                <th className="px-3 py-2">Realized</th>
                <th className="px-3 py-2">Anlik</th>
                <th className="px-3 py-2">Toplam</th>
                <th className="px-3 py-2">Kazanma</th>
              </tr>
            </thead>
            <tbody>
              {data.accounts.map(account => (
                <tr key={account.account_type} className="border-t border-gray-800">
                  <td className="px-3 py-2 font-medium">{account.label || accountLabel(account.account_type)}</td>
                  <td className="px-3 py-2 text-gray-400">{account.trade_count}</td>
                  <td className="px-3 py-2"><PnLValue value={account.realized_net_pnl} /></td>
                  <td className="px-3 py-2"><PnLValue value={account.unrealized_net_pnl} /></td>
                  <td className="px-3 py-2"><PnLValue value={account.total_net_pnl} /></td>
                  <td className="px-3 py-2 text-gray-400">{account.win_rate.toFixed(1)}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
