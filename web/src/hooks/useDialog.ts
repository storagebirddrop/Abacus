import { useState } from 'react'

/**
 * useDialog manages the open/loading/error lifecycle shared by the app's
 * form dialogs (create wallet, add price, …). It removes the duplicated
 * try/catch/finally blocks: pass the async action to `submit`, and on
 * success the dialog closes, an optional `reset` runs, and `onSuccess`
 * (typically a list reload) fires; on failure the error message is shown.
 */
export function useDialog(onSuccess?: () => void) {
  const [open, setOpen] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function submit(action: () => Promise<unknown>, reset?: () => void) {
    setError('')
    setLoading(true)
    try {
      await action()
      setOpen(false)
      reset?.()
      onSuccess?.()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  return { open, setOpen, error, loading, submit }
}
