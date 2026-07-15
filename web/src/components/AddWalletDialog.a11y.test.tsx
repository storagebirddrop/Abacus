import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { axe } from 'vitest-axe'
import { AddWalletDialog } from './AddWalletDialog'

vi.mock('../api/wallets', () => ({
  createWallet: vi.fn(),
  importWallet: vi.fn(),
}))

describe('AddWalletDialog accessibility', () => {
  it('has no axe violations when open, with a keyboard-operable drop zone and pressed-state toggle', async () => {
    const { container, findByRole } = render(
      <MemoryRouter>
        <AddWalletDialog onCreated={() => {}} />
      </MemoryRouter>,
    )

    const trigger = await findByRole('button', { name: 'Add Wallet' })
    trigger.click()

    const dropZone = await findByRole('button', { name: /upload import file/i })
    expect(dropZone).toHaveAttribute('tabIndex', '0')

    const onchainToggle = await findByRole('button', { name: 'On-chain wallet' })
    expect(onchainToggle).toHaveAttribute('aria-pressed', 'true')

    expect(await axe(container)).toHaveNoViolations()
  })
})
