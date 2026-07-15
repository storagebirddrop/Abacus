import { useEffect, useRef, useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { Menu, Moon, Sun, X } from 'lucide-react'
import { cn } from '../lib/utils'
import { useTheme } from '../hooks/useTheme'

const navItems = [
  { to: '/', label: 'Portfolio' },
  { to: '/reports', label: 'Reports' },
  { to: '/settings', label: 'Settings' },
]

export default function Layout() {
  const [open, setOpen] = useState(false)
  const { theme, toggle } = useTheme()
  const closeButtonRef = useRef<HTMLButtonElement>(null)
  const menuButtonRef = useRef<HTMLButtonElement>(null)
  const wasOpen = useRef(false)

  useEffect(() => {
    if (open) {
      wasOpen.current = true
      closeButtonRef.current?.focus()
      const onKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Escape') setOpen(false)
      }
      document.addEventListener('keydown', onKeyDown)
      return () => document.removeEventListener('keydown', onKeyDown)
    } else if (wasOpen.current) {
      wasOpen.current = false
      menuButtonRef.current?.focus()
    }
  }, [open])

  return (
    <div className="flex h-screen bg-background text-foreground">
      <a
        href="#main"
        inert={open || undefined}
        className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:m-2 focus:rounded focus:bg-primary focus:px-3 focus:py-2 focus:text-primary-foreground"
      >
        Skip to content
      </a>

      {/* Mobile top bar */}
      <div
        className="md:hidden fixed top-0 inset-x-0 z-30 flex items-center justify-between bg-card border-b border-border px-4 h-12"
        inert={open || undefined}
      >
        <span className="font-bold tracking-tight flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-primary" aria-hidden="true" />
          Abacus
        </span>
        <button
          ref={menuButtonRef}
          aria-label="Open navigation menu"
          aria-expanded={open}
          onClick={() => setOpen(true)}
          className="p-1 text-muted-foreground"
        >
          <Menu size={20} />
        </button>
      </div>

      {/* Backdrop for the mobile drawer */}
      {open && (
        <div
          className="md:hidden fixed inset-0 z-30 bg-black/60"
          onClick={() => setOpen(false)}
          aria-hidden="true"
        />
      )}

      <div
        role={open ? 'dialog' : undefined}
        aria-modal={open || undefined}
        aria-label="Primary navigation"
        className={cn(
          'bg-card border-r border-border flex flex-col z-40',
          'fixed inset-y-0 left-0 w-64 transform transition-transform md:static md:w-52 md:translate-x-0',
          open ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        <div className="px-5 py-5 border-b border-border flex items-center justify-between">
          <div>
            <span className="font-bold text-lg tracking-tight flex items-center gap-2">
              <span className="h-2 w-2 rounded-full bg-primary" aria-hidden="true" />
              Abacus
            </span>
            <p className="text-xs text-muted-foreground mt-0.5">Bitcoin Accounting</p>
          </div>
          <button
            ref={closeButtonRef}
            aria-label="Close navigation menu"
            onClick={() => setOpen(false)}
            className="md:hidden p-1 text-muted-foreground"
          >
            <X size={20} />
          </button>
        </div>
        <nav aria-label="Primary" className="flex-1 px-3 py-4 space-y-0.5">
          {navItems.map(({ to, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              onClick={() => setOpen(false)}
              className={({ isActive }) =>
                cn(
                  'block px-3 py-2 rounded-md text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-primary/15 text-primary'
                    : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
                )
              }
            >
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="px-3 py-3 border-t border-border">
          <button
            onClick={toggle}
            aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            className="flex items-center gap-2 w-full px-3 py-2 rounded-md text-sm text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
          >
            {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
            {theme === 'dark' ? 'Light mode' : 'Dark mode'}
          </button>
        </div>
      </div>

      <main id="main" className="flex-1 overflow-auto pt-12 md:pt-0" inert={open || undefined}>
        <Outlet />
      </main>
    </div>
  )
}
