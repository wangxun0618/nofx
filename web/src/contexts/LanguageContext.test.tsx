import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { LanguageProvider, useLanguage } from './LanguageContext'
import { LanguageSwitcher } from '../components/common/LanguageSwitcher'
import { t } from '../i18n/translations'

function LanguageProbe() {
  const { language, setLanguage } = useLanguage()
  return (
    <div>
      <span data-testid="current">{language}</span>
      <button type="button" onClick={() => setLanguage('en')}>
        to-en
      </button>
    </div>
  )
}

describe('bilingual language policy', () => {
  beforeEach(() => localStorage.clear())

  it('defaults to Chinese so the primary market lands in its own language', () => {
    render(
      <LanguageProvider>
        <LanguageProbe />
      </LanguageProvider>
    )

    expect(screen.getByTestId('current').textContent).toBe('zh')
  })

  it('restores an explicit choice made on a previous visit', () => {
    localStorage.setItem('language', 'en')

    render(
      <LanguageProvider>
        <LanguageProbe />
      </LanguageProvider>
    )

    expect(screen.getByTestId('current').textContent).toBe('en')
  })

  it('defaults to Chinese even when the browser prefers English', () => {
    const original = window.navigator.languages
    Object.defineProperty(window.navigator, 'languages', {
      value: ['en-US', 'en'],
      configurable: true,
    })

    render(
      <LanguageProvider>
        <LanguageProbe />
      </LanguageProvider>
    )

    expect(screen.getByTestId('current').textContent).toBe('zh')
    Object.defineProperty(window.navigator, 'languages', {
      value: original,
      configurable: true,
    })
  })

  it('persists a switch and keeps <html lang> in sync', async () => {
    render(
      <LanguageProvider>
        <LanguageProbe />
      </LanguageProvider>
    )

    fireEvent.click(screen.getByText('to-en'))

    await waitFor(() => {
      expect(screen.getByTestId('current').textContent).toBe('en')
      expect(localStorage.getItem('language')).toBe('en')
      expect(document.documentElement.getAttribute('lang')).toBe('en')
    })
  })

  it('renders a switcher offering exactly the two supported languages', () => {
    render(
      <LanguageProvider>
        <LanguageSwitcher />
      </LanguageProvider>
    )

    expect(screen.getByRole('button', { name: t('lang.zh', 'zh') })).toBeTruthy()
    expect(screen.getByRole('button', { name: t('lang.en', 'zh') })).toBeTruthy()
    // Indonesian was dropped: the product ships Chinese and English only.
    expect(screen.queryByRole('button', { name: 'ID' })).toBeNull()
  })

  it('marks the active language as pressed and switches on click', async () => {
    render(
      <LanguageProvider>
        <LanguageSwitcher />
      </LanguageProvider>
    )

    const zhButton = screen.getByRole('button', { name: t('lang.zh', 'zh') })
    const enButton = screen.getByRole('button', { name: t('lang.en', 'zh') })
    expect(zhButton.getAttribute('aria-pressed')).toBe('true')
    expect(enButton.getAttribute('aria-pressed')).toBe('false')

    fireEvent.click(enButton)

    await waitFor(() =>
      expect(enButton.getAttribute('aria-pressed')).toBe('true')
    )
  })
})
