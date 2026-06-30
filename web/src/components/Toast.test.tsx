import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ToastProvider, useToast } from './Toast'

function Trigger() {
  const { toast } = useToast()
  return <button onClick={() => toast('Saved!', 'success')}>fire</button>
}

describe('Toast', () => {
  it('shows a toast when triggered and dismisses on click', async () => {
    render(
      <ToastProvider>
        <Trigger />
      </ToastProvider>,
    )
    await userEvent.click(screen.getByRole('button', { name: 'fire' }))

    const toast = await screen.findByText('Saved!')
    expect(toast).toBeInTheDocument()

    await userEvent.click(toast)
    expect(screen.queryByText('Saved!')).not.toBeInTheDocument()
  })

  it('throws if used outside the provider', () => {
    function Bad() {
      useToast()
      return null
    }
    // Suppress React's error logging for this expected throw.
    expect(() => render(<Bad />)).toThrow(/ToastProvider/)
  })
})
