import { createContext, useCallback, useContext, useState, type ReactNode } from 'react'
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

  const remove = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const toast = useCallback(
    (message: string, variant: Variant = 'info') => {
      const id = nextID++
      setToasts((prev) => [...prev, { id, message, variant }])
      setTimeout(() => remove(id), 4000)
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
              'rounded-md px-4 py-2 text-sm shadow-lg max-w-sm cursor-pointer',
              t.variant === 'success' && 'bg-green-600 text-white',
              t.variant === 'error' && 'bg-red-600 text-white',
              t.variant === 'info' && 'bg-slate-800 text-white',
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
