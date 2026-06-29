import { NavLink, Outlet } from 'react-router-dom'
import { cn } from '../lib/utils'

const navItems = [
  { to: '/wallets', label: 'Wallets' },
  { to: '/prices', label: 'Prices' },
]

export default function Layout() {
  return (
    <div className="flex h-screen bg-slate-50">
      <aside className="w-48 bg-slate-900 text-white flex flex-col">
        <div className="px-4 py-5 border-b border-slate-700">
          <span className="font-bold text-lg tracking-tight">Abacus</span>
          <p className="text-xs text-slate-400 mt-0.5">Bitcoin Accounting</p>
        </div>
        <nav className="flex-1 px-2 py-4 space-y-1">
          {navItems.map(({ to, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                cn(
                  'block px-3 py-2 rounded-md text-sm transition-colors',
                  isActive
                    ? 'bg-slate-700 text-white'
                    : 'text-slate-300 hover:bg-slate-800 hover:text-white'
                )
              }
            >
              {label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}
