import { useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { Menu, Moon, Sun, X } from 'lucide-react'
import { cn } from '../lib/utils'
import { useTheme } from '../hooks/useTheme'

const navItems = [
  { to: '/wallets', label: 'Wallets' },
  { to: '/prices', label: 'Prices' },
  { to: '/settings', label: 'Settings' },
]

export default function Layout() {
  const [open, setOpen] = useState(false)
  const { theme, toggle } = useTheme()

  return (
    <div className="flex h-screen bg-slate-50 dark:bg-slate-950">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:m-2 focus:rounded focus:bg-slate-900 focus:px-3 focus:py-2 focus:text-white"
      >
        Skip to content
      </a>

      {/* Mobile top bar */}
      <div className="md:hidden fixed top-0 inset-x-0 z-30 flex items-center justify-between bg-slate-900 text-white px-4 h-12">
        <span className="font-bold tracking-tight">Abacus</span>
        <button
          aria-label="Open navigation menu"
          aria-expanded={open}
          onClick={() => setOpen(true)}
          className="p-1"
        >
          <Menu size={20} />
        </button>
      </div>

      {/* Backdrop for the mobile drawer */}
      {open && (
        <div
          className="md:hidden fixed inset-0 z-30 bg-black/40"
          onClick={() => setOpen(false)}
          aria-hidden="true"
        />
      )}

      <aside
        className={cn(
          'bg-slate-900 text-white flex flex-col z-40',
          'fixed inset-y-0 left-0 w-64 transform transition-transform md:static md:w-48 md:translate-x-0',
          open ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        <div className="px-4 py-5 border-b border-slate-700 flex items-center justify-between">
          <div>
            <span className="font-bold text-lg tracking-tight">Abacus</span>
            <p className="text-xs text-slate-400 mt-0.5">Bitcoin Accounting</p>
          </div>
          <button
            aria-label="Close navigation menu"
            onClick={() => setOpen(false)}
            className="md:hidden p-1"
          >
            <X size={20} />
          </button>
        </div>
        <nav aria-label="Primary" className="flex-1 px-2 py-4 space-y-1">
          {navItems.map(({ to, label }) => (
            <NavLink
              key={to}
              to={to}
              onClick={() => setOpen(false)}
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
        <div className="px-2 py-3 border-t border-slate-700">
          <button
            onClick={toggle}
            aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            className="flex items-center gap-2 w-full px-3 py-2 rounded-md text-sm text-slate-300 hover:bg-slate-800 hover:text-white"
          >
            {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
            {theme === 'dark' ? 'Light mode' : 'Dark mode'}
          </button>
        </div>
      </aside>

      <main id="main" className="flex-1 overflow-auto pt-12 md:pt-0">
        <Outlet />
      </main>
    </div>
  )
}
