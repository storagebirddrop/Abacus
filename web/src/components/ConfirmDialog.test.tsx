import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ConfirmProvider, useConfirm } from './ConfirmDialog'

function Trigger({ onResult }: { onResult: (v: boolean) => void }) {
  const confirm = useConfirm()
  return (
    <button
      onClick={async () => {
        const ok = await confirm({ title: 'Delete wallet', confirmText: 'Delete', destructive: true })
        onResult(ok)
      }}
    >
      go
    </button>
  )
}

describe('ConfirmDialog', () => {
  it('resolves true when confirmed', async () => {
    let result: boolean | undefined
    render(
      <ConfirmProvider>
        <Trigger onResult={(v) => (result = v)} />
      </ConfirmProvider>,
    )
    await userEvent.click(screen.getByRole('button', { name: 'go' }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
    expect(result).toBe(true)
  })

  it('resolves false when cancelled', async () => {
    let result: boolean | undefined
    render(
      <ConfirmProvider>
        <Trigger onResult={(v) => (result = v)} />
      </ConfirmProvider>,
    )
    await userEvent.click(screen.getByRole('button', { name: 'go' }))
    const dialog = await screen.findByRole('dialog')
    await userEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    expect(result).toBe(false)
  })
})
