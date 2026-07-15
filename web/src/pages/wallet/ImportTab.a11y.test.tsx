import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import { render } from '@testing-library/react'
import { axe } from 'vitest-axe'

vi.mock('../../api/wallets', () => ({
  importWallet: vi.fn(),
  getImportJob: vi.fn(),
  listImportJobs: vi.fn(),
}))

import { listImportJobs } from '../../api/wallets'
import { ImportTab } from './ImportTab'

const listImportMock = listImportJobs as unknown as Mock

beforeEach(() => {
  listImportMock.mockReset()
  listImportMock.mockResolvedValue([])
})

describe('ImportTab accessibility', () => {
  it('has no axe violations and exposes the drop zone as a keyboard-operable button', async () => {
    const { container, findByRole } = render(<ImportTab walletID="w1" />)
    const dropZone = await findByRole('button', { name: /upload wallet export file/i })
    expect(dropZone).toHaveAttribute('tabIndex', '0')

    expect(await axe(container)).toHaveNoViolations()
  })
})
