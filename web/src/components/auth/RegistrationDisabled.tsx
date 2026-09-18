import { useNavigate } from 'react-router-dom'
import { useLanguage } from '../../contexts/LanguageContext'
import { t } from '../../i18n/translations'
import { LanguageSwitcher } from '../common/LanguageSwitcher'

export function RegistrationDisabled() {
  const { language } = useLanguage()
  const navigate = useNavigate()

  const handleBackToLogin = () => {
    navigate('/login')
  }

  return (
    <div
      className="relative min-h-screen flex items-center justify-center"
      style={{ background: '#F1ECE2', color: '#1A1813' }}
    >
      <LanguageSwitcher />
      <div className="text-center max-w-md px-6">
        <img
          src="/icons/nofx.svg"
          alt={t('auth.noFxLogo', language)}
          className="w-16 h-16 mx-auto mb-4"
        />
        <h1 className="text-2xl font-semibold mb-3">
          {t('registrationClosed', language)}
        </h1>
        <p className="text-sm text-nofx-text-muted">
          {t('registrationClosedMessage', language)}
        </p>
        <button
          className="mt-6 px-4 py-2 rounded text-sm font-semibold transition-colors hover:opacity-90"
          style={{ background: '#E0483B', color: '#F1ECE2' }}
          onClick={handleBackToLogin}
        >
          {t('backToLogin', language)}
        </button>
      </div>
    </div>
  )
}
