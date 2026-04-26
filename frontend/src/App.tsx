import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Webhooks from './pages/Webhooks';
import Trades from './pages/Trades';
import PnLAnalysis from './pages/PnLAnalysis';
import SystemStatus from './pages/SystemStatus';
import Settings from './pages/Settings';

const navItems = [
  { to: '/', label: 'Dashboard' },
  { to: '/webhooks', label: 'Gelen Istekler' },
  { to: '/trades', label: 'Islemler' },
  { to: '/pnl', label: 'Kar/Zarar' },
  { to: '/system', label: 'Sistem Durumu' },
  { to: '/settings', label: 'Ayarlar' },
];

export default function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-gray-950 text-gray-100">
        <nav className="bg-gray-900 border-b border-gray-800 px-6 py-3">
          <div className="max-w-7xl mx-auto flex items-center gap-8">
            <span className="text-lg font-bold text-emerald-400 tracking-tight">TradingBot V2</span>
            <div className="flex gap-1">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.to === '/'}
                  className={({ isActive }) =>
                    `px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-emerald-600/20 text-emerald-400'
                        : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800'
                    }`
                  }
                >
                  {item.label}
                </NavLink>
              ))}
            </div>
          </div>
        </nav>

        <main className="max-w-7xl mx-auto px-6 py-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/webhooks" element={<Webhooks />} />
            <Route path="/trades" element={<Trades />} />
            <Route path="/pnl" element={<PnLAnalysis />} />
            <Route path="/system" element={<SystemStatus />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}
