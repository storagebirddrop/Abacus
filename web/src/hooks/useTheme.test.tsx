import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useTheme } from './useTheme'

function Probe() {
  const { theme, toggle } = useTheme()
  return (
    <div>
      <span data-testid="theme">{theme}</span>
      <button onClick={toggle}>toggle</button>
    </div>
  )
}

beforeEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('light')
})
afterEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove('light')
})

describe('useTheme', () => {
  it('defaults to dark and toggles to light, applying the class + persisting', async () => {
    render(<Probe />)
    expect(screen.getByTestId('theme')).toHaveTextContent('dark')
    expect(document.documentElement.classList.contains('light')).toBe(false)

    await userEvent.click(screen.getByRole('button', { name: 'toggle' }))

    expect(screen.getByTestId('theme')).toHaveTextContent('light')
    expect(document.documentElement.classList.contains('light')).toBe(true)
    expect(localStorage.getItem('abacus-theme')).toBe('light')
  })

  it('restores a persisted light theme', () => {
    localStorage.setItem('abacus-theme', 'light')
    render(<Probe />)
    expect(screen.getByTestId('theme')).toHaveTextContent('light')
    expect(document.documentElement.classList.contains('light')).toBe(true)
  })
})
