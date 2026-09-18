import { Globe } from 'lucide-react'
import { useLanguage } from '../../contexts/LanguageContext'
import { t, type Language } from '../../i18n/translations'

const LANGUAGES: { code: Language; labelKey: string }[] = [
  { code: 'zh', labelKey: 'lang.zh' },
  { code: 'en', labelKey: 'lang.en' },
]

interface LanguageSwitcherProps {
  /**
   * `overlay` floats at the top-right of pages without a header bar
   * (landing / login / register). `inline` sits inside a header bar.
   */
  variant?: 'overlay' | 'inline'
  className?: string
}

export function LanguageSwitcher({
  variant = 'overlay',
  className = '',
}: LanguageSwitcherProps) {
  const { language, setLanguage } = useLanguage()

  const shell =
    variant === 'overlay'
      ? 'absolute top-4 right-4 z-50 border border-[rgba(26,24,19,0.14)] bg-nofx-bg-lighter'
      : 'border border-[rgba(26,24,19,0.14)] bg-nofx-bg-deeper'

  return (
    <div
      role="group"
      aria-label={t('lang.label', language)}
      className={`flex items-center gap-0.5 rounded-lg p-0.5 backdrop-blur-sm ${shell} ${className}`}
    >
      <Globe
        size={13}
        className="ml-1.5 mr-0.5 shrink-0 text-nofx-text-muted"
        aria-hidden="true"
      />
      {LANGUAGES.map(({ code, labelKey }) => (
        <button
          key={code}
          type="button"
          onClick={() => setLanguage(code)}
          aria-pressed={language === code}
          className={`whitespace-nowrap rounded px-2 py-1 text-xs font-semibold transition-all ${
            language === code
              ? 'bg-nofx-gold/15 text-nofx-gold'
              : 'bg-transparent text-nofx-text-muted hover:text-nofx-text'
          }`}
        >
          {t(labelKey, language)}
        </button>
      ))}
    </div>
  )
}
