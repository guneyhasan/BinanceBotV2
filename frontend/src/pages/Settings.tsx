import { useEffect, useState } from 'react';
import { api } from '../api/client';
import type { Config } from '../types';

export default function Settings() {
  const [config, setConfig] = useState<Config | null>(null);
  const [form, setForm] = useState({ trade_amount_usd: '', leverage: '', margin_mode: '', commission_rate: '' });
  const [saving, setSaving] = useState(false);
  const [telegramTesting, setTelegramTesting] = useState<'signal' | 'trade' | null>(null);
  const [message, setMessage] = useState('');

  useEffect(() => {
    api.getConfig().then(cfg => {
      setConfig(cfg);
      setForm({
        trade_amount_usd: String(cfg.trade_amount_usd),
        leverage: String(cfg.leverage),
        margin_mode: cfg.margin_mode,
        commission_rate: String(cfg.commission_rate),
      });
    });
  }, []);

  const save = async () => {
    setSaving(true);
    setMessage('');
    try {
      const data: Record<string, unknown> = {};
      if (form.trade_amount_usd) data.trade_amount_usd = parseFloat(form.trade_amount_usd);
      if (form.leverage) data.leverage = parseInt(form.leverage);
      if (form.margin_mode) data.margin_mode = form.margin_mode;
      if (form.commission_rate) data.commission_rate = parseFloat(form.commission_rate);

      const updated = await api.updateConfig(data as Partial<Config>);
      setConfig(updated);
      setMessage('Ayarlar kaydedildi');
    } catch (err) {
      setMessage('Hata: ' + (err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const testTelegram = async (target: 'signal' | 'trade') => {
    setTelegramTesting(target);
    setMessage('');
    try {
      await api.testTelegram(target);
      setMessage(target === 'signal' ? 'Telegram sinyal test mesaji gonderildi' : 'Telegram islem test mesaji gonderildi');
    } catch (err) {
      setMessage('Hata: ' + (err as Error).message);
    } finally {
      setTelegramTesting(null);
    }
  };

  if (!config) return <div className="text-gray-500 py-8">Yukleniyor...</div>;

  return (
    <div className="space-y-6 max-w-lg">
      <h1 className="text-2xl font-bold">Ayarlar</h1>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6 space-y-5">
        <div>
          <label className="block text-sm text-gray-400 mb-1">Islem Tutari (USD)</label>
          <input
            type="number"
            step="0.01"
            className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            value={form.trade_amount_usd}
            onChange={e => setForm(f => ({ ...f, trade_amount_usd: e.target.value }))}
          />
          <p className="text-xs text-gray-600 mt-1">Her islemde kullanilacak USD tutari</p>
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">Kaldırac (Leverage)</label>
          <input
            type="number"
            step="1"
            min="1"
            max="125"
            className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            value={form.leverage}
            onChange={e => setForm(f => ({ ...f, leverage: e.target.value }))}
          />
          <p className="text-xs text-gray-600 mt-1">Binance Futures kaldırac degeri (1-125x)</p>
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">Margin Modu</label>
          <select
            className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            value={form.margin_mode}
            onChange={e => setForm(f => ({ ...f, margin_mode: e.target.value }))}
          >
            <option value="ISOLATED">ISOLATED (Izole)</option>
            <option value="CROSSED">CROSSED (Capraz)</option>
          </select>
        </div>

        <div>
          <label className="block text-sm text-gray-400 mb-1">Komisyon Orani</label>
          <input
            type="number"
            step="0.000001"
            className="w-full bg-gray-800 border border-gray-700 rounded px-3 py-2 text-sm"
            value={form.commission_rate}
            onChange={e => setForm(f => ({ ...f, commission_rate: e.target.value }))}
          />
          <p className="text-xs text-gray-600 mt-1">Binance taker komisyonu (varsayilan: 0.0004 = %0.04)</p>
        </div>

        <button
          onClick={save}
          disabled={saving}
          className="w-full bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white font-medium py-2.5 rounded transition-colors"
        >
          {saving ? 'Kaydediliyor...' : 'Kaydet'}
        </button>

        {message && (
          <div className={`text-sm text-center ${message.startsWith('Hata') ? 'text-red-400' : 'text-emerald-400'}`}>
            {message}
          </div>
        )}
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-3">Mevcut Yapilandirma</h2>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div className="text-gray-500">Islem Tutari:</div>
          <div className="font-mono">{config.trade_amount_usd} USD</div>
          <div className="text-gray-500">Kaldırac:</div>
          <div className="font-mono">{config.leverage}x</div>
          <div className="text-gray-500">Margin Modu:</div>
          <div>{config.margin_mode}</div>
          <div className="text-gray-500">Komisyon Orani:</div>
          <div className="font-mono">{config.commission_rate} (%{(config.commission_rate * 100).toFixed(2)})</div>
          <div className="text-gray-500">Son Guncelleme:</div>
          <div className="text-xs text-gray-400">{config.updated_at}</div>
        </div>
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-6 space-y-4">
        <div>
          <h2 className="text-lg font-semibold">Telegram Testi</h2>
          <p className="text-sm text-gray-500 mt-1">
            Bot token ve chat ID ayarlarini kontrol etmek icin test bildirimi gonderin.
          </p>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <button
            onClick={() => testTelegram('signal')}
            disabled={telegramTesting !== null}
            className="bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white font-medium py-2.5 rounded transition-colors"
          >
            {telegramTesting === 'signal' ? 'Gonderiliyor...' : 'Sinyal Chat Testi'}
          </button>
          <button
            onClick={() => testTelegram('trade')}
            disabled={telegramTesting !== null}
            className="bg-purple-600 hover:bg-purple-500 disabled:opacity-50 text-white font-medium py-2.5 rounded transition-colors"
          >
            {telegramTesting === 'trade' ? 'Gonderiliyor...' : 'Islem Chat Testi'}
          </button>
        </div>
      </div>
    </div>
  );
}
