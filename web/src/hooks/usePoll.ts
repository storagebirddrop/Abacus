import { useCallback, useEffect, useRef } from 'react'

/**
 * usePoll manages a single setInterval lifecycle: start a polling callback,
 * stop it, and auto-clear on unmount. Replaces the hand-rolled pollRef pattern.
 *
 * The callback is invoked every `intervalMs`; call `stop()` from inside it (or
 * externally) to end polling — e.g. when a job reaches a terminal state.
 */
export function usePoll() {
  const ref = useRef<ReturnType<typeof setInterval> | null>(null)

  const stop = useCallback(() => {
    if (ref.current) {
      clearInterval(ref.current)
      ref.current = null
    }
  }, [])

  const start = useCallback(
    (fn: () => void | Promise<void>, intervalMs: number) => {
      stop()
      ref.current = setInterval(fn, intervalMs)
    },
    [stop],
  )

  // Clear any active interval when the consuming component unmounts.
  useEffect(() => stop, [stop])

  return { start, stop }
}
