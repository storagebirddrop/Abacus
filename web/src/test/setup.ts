import '@testing-library/jest-dom'
import { afterEach, expect } from 'vitest'
import { cleanup } from '@testing-library/react'
import * as axeMatchers from 'vitest-axe/matchers'

expect.extend(axeMatchers)

// Unmount React trees between tests so DOM state doesn't leak across cases.
afterEach(() => {
  cleanup()
})

// jsdom lacks a few DOM APIs that Radix UI primitives (Dialog/Select) touch.
// Provide no-op shims so those components can render in tests.
if (typeof Element !== 'undefined') {
  Element.prototype.hasPointerCapture ??= () => false
  Element.prototype.setPointerCapture ??= () => {}
  Element.prototype.releasePointerCapture ??= () => {}
  Element.prototype.scrollIntoView ??= () => {}
}
if (typeof globalThis.ResizeObserver === 'undefined') {
  globalThis.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}
