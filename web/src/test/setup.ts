import '@testing-library/jest-dom'
import { beforeAll, afterEach } from 'vitest'

// Mock localStorage
const localStorageMock = {
  getItem: (key: string) => {
    return localStorageMock._store[key] || null
  },
  setItem: (key: string, value: string) => {
    localStorageMock._store[key] = value
  },
  removeItem: (key: string) => {
    delete localStorageMock._store[key]
  },
  clear: () => {
    localStorageMock._store = {}
  },
  _store: {} as Record<string, string>,
}

// Setup before all tests
beforeAll(() => {
  Object.defineProperty(window, 'localStorage', {
    value: localStorageMock,
    writable: true,
  })
})

// framer-motion's `whileInView` features need an IntersectionObserver, which
// jsdom does not implement. Without this, any component using scroll-triggered
// animation throws on mount and cannot be rendered in tests at all.
if (!('IntersectionObserver' in window)) {
  class IntersectionObserverStub implements IntersectionObserver {
    readonly root: Element | Document | null = null
    readonly rootMargin: string = ''
    readonly thresholds: ReadonlyArray<number> = []
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords(): IntersectionObserverEntry[] {
      return []
    }
  }
  Object.defineProperty(window, 'IntersectionObserver', {
    value: IntersectionObserverStub,
    writable: true,
  })
}

// Clean up after each test
afterEach(() => {
  localStorageMock.clear()
})
