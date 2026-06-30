import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { AppSettings } from '../api/settings'

// Mock the settings API module so the page renders without a backend.
const getSettings = vi.fn()
const updateSettings = vi.fn()
vi.mock('../api/settings', () => ({
  getSettings: () => getSettings(),
  updateSettings: (data: Partial<AppSettings>) => updateSettings(data),
}))

import SettingsPage from './SettingsPage'

const baseSettings: AppSettings = {
  sync_enabled: false,
  blockchain_backend: 'esplora',
  esplora_url: 'https://mempool.space/api',
  esplora_rate_ms: 100,
  electrum_host: 'electrum.blockstream.info',
  electrum_port: 50002,
  electrum_tls: true,
}

beforeEach(() => {
  getSettings.mockReset()
  updateSettings.mockReset()
})

describe('SettingsPage privacy notice', () => {
  // The notice body lives in the text node alongside a "Privacy notice:" span,
  // so we match on the body sentence to land on the containing <p>.
  const noticeBody = /syncing queries the server below/i

  it('hides the privacy notice while sync is disabled', async () => {
    getSettings.mockResolvedValue({ ...baseSettings, sync_enabled: false })
    render(<SettingsPage />)

    // Wait for load to finish (checkbox appears).
    await screen.findByText('Enable blockchain sync')
    expect(screen.queryByText(noticeBody)).not.toBeInTheDocument()
  })

  it('shows the address-disclosure privacy notice once sync is enabled', async () => {
    getSettings.mockResolvedValue({ ...baseSettings, sync_enabled: false })
    render(<SettingsPage />)

    const checkbox = await screen.findByRole('checkbox', { name: /enable blockchain sync/i })
    await userEvent.click(checkbox)

    const notice = await screen.findByText(noticeBody)
    expect(notice).toBeInTheDocument()
    expect(notice.textContent).toMatch(/addresses/i)
    expect(notice.textContent).toMatch(/host (it )?yourself/i)
  })

  it('renders an error when settings fail to load', async () => {
    getSettings.mockRejectedValue(new Error('boom'))
    render(<SettingsPage />)
    await waitFor(() => expect(screen.getByText('boom')).toBeInTheDocument())
  })
})
