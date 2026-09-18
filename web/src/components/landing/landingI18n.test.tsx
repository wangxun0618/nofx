import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { LanguageProvider } from '../../contexts/LanguageContext'
import { LanguageSwitcher } from '../common/LanguageSwitcher'
import DeploymentHub from './core/DeploymentHub'
import BrandFeatures from './brand/BrandFeatures'
import FooterSection from './FooterSection'

/**
 * Integration test for the restored language switcher.
 *
 * The unit tests in `contexts/LanguageContext.test.tsx` cover the state layer.
 * This file proves the other half of the contract: real sections of the app
 * render localized copy and re-render in the other language when the switcher
 * is used — i.e. the copy is genuinely wired through `t()`, not left as
 * hardcoded English behind a switcher that only changes a variable.
 */

function renderWithLang(ui: React.ReactNode) {
  return render(<LanguageProvider>{ui}</LanguageProvider>)
}

describe('landing page localization', () => {
  beforeEach(() => localStorage.clear())

  it('renders the deployment section in Chinese by default', () => {
    renderWithLang(<DeploymentHub />)

    expect(screen.getByText('系统部署')).toBeTruthy()
    expect(screen.getByText('一行命令安装')).toBeTruthy()
    expect(screen.getByText('密钥留在本地')).toBeTruthy()
    expect(screen.queryByText('One-Line Install')).toBeNull()
  })

  it('renders the same section in English when a prior visit chose English', () => {
    localStorage.setItem('language', 'en')

    renderWithLang(<DeploymentHub />)

    expect(screen.getByText('One-Line Install')).toBeTruthy()
    expect(screen.getByText('Your Keys Stay Home')).toBeTruthy()
    expect(screen.queryByText('一行命令安装')).toBeNull()
  })

  it('swaps the copy live when the switcher is used', async () => {
    renderWithLang(
      <>
        <LanguageSwitcher />
        <DeploymentHub />
      </>
    )

    // Chinese first…
    expect(screen.getByText('系统部署')).toBeTruthy()

    // …then flip to English and back.
    fireEvent.click(screen.getByRole('button', { name: 'EN' }))
    await waitFor(() => expect(screen.getByText('System Deployment')).toBeTruthy())
    expect(screen.queryByText('系统部署')).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: '中文' }))
    await waitFor(() => expect(screen.getByText('系统部署')).toBeTruthy())
    expect(screen.queryByText('System Deployment')).toBeNull()
  })

  it('localizes the brand feature grid', async () => {
    renderWithLang(<BrandFeatures />)

    // Translated taglines, not the English source strings.
    expect(screen.queryByText('AI DRIVEN')).toBeNull()
    expect(screen.getByText('开源')).toBeTruthy()
  })

  it('localizes a component that takes an explicit language prop', () => {
    function Harness() {
      return <FooterSection language="zh" />
    }

    renderWithLang(<Harness />)

    // `Issues` / `Pull Requests` are translated; brand names stay as-is.
    expect(screen.queryByText('Issues')).toBeNull()
    expect(screen.getByText('问题反馈')).toBeTruthy()
  })
})
