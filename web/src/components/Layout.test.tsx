import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { axe } from 'vitest-axe'
import Layout from './Layout'

function renderLayout() {
  return render(
    <MemoryRouter initialEntries={['/']}>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<div>Home content</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  )
}

describe('Layout mobile drawer', () => {
  it('has no axe violations when closed', async () => {
    const { container } = renderLayout()
    expect(await axe(container)).toHaveNoViolations()
  })

  it('opens as a modal dialog, moves focus in, inerts the main content, and closes on Escape', async () => {
    const user = userEvent.setup()
    renderLayout()

    const openButton = screen.getByRole('button', { name: 'Open navigation menu' })
    await user.click(openButton)

    const closeButton = await screen.findByRole('button', { name: 'Close navigation menu' })
    expect(closeButton).toHaveFocus()

    const main = document.getElementById('main')
    expect(main).toHaveAttribute('inert')

    await user.keyboard('{Escape}')

    expect(openButton).toHaveFocus()
    expect(document.getElementById('main')).not.toHaveAttribute('inert')
  })
})
