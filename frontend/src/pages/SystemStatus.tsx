import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { SystemStats } from '../types';
import Card from '../components/Card';
import StatusBadge from '../components/StatusBadge';

export default function SystemStatus() {
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [loading, setLoading] = useState(true);

  const load = () => {
    setLoading(true);
    api.getSystemStats().then(setStats).finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, []);

  if (loading) return <div className="text-gray-500 py-8">Yukleniyor...</div>;
  if (!stats) return <div className="text-red-400 py-8">Veri alinamadi</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Sistem Durumu</h1>
        <button onClick={load} className="bg-gray-700 hover:bg-gray-600 px-3 py-1.5 rounded text-sm">Yenile</button>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        <Card title="Toplam Istek" value={stats.total_webhooks} color="blue" />
        <Card
          title="Basari Orani"
          value={`%${stats.success_rate.toFixed(1)}`}
          subtitle={`${stats.completed_count + stats.retry_success_count} basarili`}
          color="green"
        />
        <Card
          title="Retry Orani"
          value={`%${stats.retry_rate.toFixed(1)}`}
          subtitle={`Ort. ${stats.avg_retry_count.toFixed(1)} deneme`}
          color="yellow"
        />
        <Card
          title="Basarisiz"
          value={`%${stats.fail_rate.toFixed(1)}`}
          subtitle={`${stats.failed_count} istek`}
          color="red"
        />
        <Card
          title="Isleniyor"
          value={stats.processing_count}
          color="blue"
        />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
          <h2 className="text-lg font-semibold mb-4">Hata Dagilimi</h2>
          {!stats.error_breakdown || stats.error_breakdown.length === 0 ? (
            <p className="text-gray-500 text-sm">Hata kaydedilmemis</p>
          ) : (
            <div className="space-y-2">
              {stats.error_breakdown.map(e => {
                const total = stats.error_breakdown.reduce((s, x) => s + x.count, 0);
                const pct = total > 0 ? (e.count / total * 100) : 0;
                return (
                  <div key={e.category} className="space-y-1">
                    <div className="flex justify-between text-sm">
                      <span>{e.category}</span>
                      <span className="text-gray-400">{e.count} ({pct.toFixed(0)}%)</span>
                    </div>
                    <div className="w-full bg-gray-800 rounded-full h-2">
                      <div className="bg-red-500 h-2 rounded-full" style={{ width: `${pct}%` }} />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
          <h2 className="text-lg font-semibold mb-4">Islem Sonuc Dagilimi</h2>
          <div className="space-y-3">
            {[
              { label: 'Direkt Basarili', count: stats.completed_count, color: 'bg-emerald-500' },
              { label: 'Retry Sonrasi Basarili', count: stats.retry_success_count, color: 'bg-yellow-500' },
              { label: 'Basarisiz', count: stats.failed_count, color: 'bg-red-500' },
              { label: 'Isleniyor', count: stats.processing_count, color: 'bg-blue-500' },
            ].map(item => {
              const pct = stats.total_webhooks > 0 ? (item.count / stats.total_webhooks * 100) : 0;
              return (
                <div key={item.label} className="space-y-1">
                  <div className="flex justify-between text-sm">
                    <span>{item.label}</span>
                    <span className="text-gray-400">{item.count} ({pct.toFixed(1)}%)</span>
                  </div>
                  <div className="w-full bg-gray-800 rounded-full h-2">
                    <div className={`${item.color} h-2 rounded-full`} style={{ width: `${pct}%` }} />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-5">
        <h2 className="text-lg font-semibold mb-4">Kesinlikle Basarisiz Islemler</h2>
        {!stats.failed_details || stats.failed_details.length === 0 ? (
          <p className="text-emerald-400 text-sm">Kalici hata yok</p>
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
                  <th className="pb-2">Hata</th>
                </tr>
              </thead>
              <tbody>
                {stats.failed_details.map(w => (
                  <tr key={w.id} className="border-b border-gray-800/50">
                    <td className="py-2 pr-4 text-gray-400 font-mono text-xs">{new Date(w.received_at).toLocaleString('tr-TR')}</td>
                    <td className="py-2 pr-4 font-medium">{w.coin}</td>
                    <td className="py-2 pr-4">{w.signal_type}</td>
                    <td className="py-2 pr-4"><StatusBadge status={w.status} /></td>
                    <td className="py-2 pr-4">{w.retry_count}</td>
                    <td className="py-2 text-red-400 text-xs max-w-xs truncate">{w.error_message || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
