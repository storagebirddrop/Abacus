import { describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useDialog } from './useDialog'

describe('useDialog', () => {
  it('closes, resets, and fires onSuccess after a successful submit', async () => {
    const onSuccess = vi.fn()
    const reset = vi.fn()
    const { result } = renderHook(() => useDialog(onSuccess))

    act(() => result.current.setOpen(true))
    expect(result.current.open).toBe(true)

    await act(async () => {
      await result.current.submit(async () => {}, reset)
    })

    expect(result.current.open).toBe(false)
    expect(result.current.error).toBe('')
    expect(result.current.loading).toBe(false)
    expect(reset).toHaveBeenCalledOnce()
    expect(onSuccess).toHaveBeenCalledOnce()
  })

  it('surfaces the error message and stays open when the action throws', async () => {
    const onSuccess = vi.fn()
    const reset = vi.fn()
    const { result } = renderHook(() => useDialog(onSuccess))

    act(() => result.current.setOpen(true))

    await act(async () => {
      await result.current.submit(async () => {
        throw new Error('boom')
      }, reset)
    })

    expect(result.current.open).toBe(true)
    expect(result.current.error).toBe('boom')
    expect(result.current.loading).toBe(false)
    expect(reset).not.toHaveBeenCalled()
    expect(onSuccess).not.toHaveBeenCalled()
  })
})
