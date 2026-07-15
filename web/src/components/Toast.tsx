import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'
import { cn } from '../lib/utils'

type Variant = 'success' | 'error' | 'info'

interface Toast {
  id: number
  message: string
  variant: Variant
}

interface ToastContextValue {
  toast: (message: string, variant?: Variant) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

// eslint-disable-next-line react-refresh/only-export-components
export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within <ToastProvider>')
  return ctx
}

let nextID = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const timers = useRef<Set<ReturnType<typeof setTimeout>>>(new Set())

  useEffect(() => {
    const active = timers.current
    return () => {
      active.forEach((t) => clearTimeout(t))
      active.clear()
    }
  }, [])

  const remove = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const toast = useCallback(
    (message: string, variant: Variant = 'info') => {
      const id = nextID++
      setToasts((prev) => [...prev, { id, message, variant }])
      const t = setTimeout(() => {
        timers.current.delete(t)
        remove(id)
      }, 4000)
      timers.current.add(t)
    },
    [remove],
  )

  return (
    <ToastContext.Provider value={{ toast }}>
      {children}
      <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2" role="region" aria-label="Notifications">
        {toasts.map((t) => (
          <div
            key={t.id}
            role="status"
            className={cn(
              'rounded-md px-4 py-2 text-sm shadow-lg border border-border max-w-sm cursor-pointer',
              t.variant === 'success' && 'bg-success text-success-foreground border-transparent',
              t.variant === 'error' && 'bg-destructive text-destructive-foreground border-transparent',
              t.variant === 'info' && 'bg-card text-card-foreground',
            )}
            onClick={() => remove(t.id)}
          >
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
