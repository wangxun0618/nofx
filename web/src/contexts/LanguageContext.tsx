import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import type { Language } from '../i18n/translations'
import { setActiveLanguage } from '../i18n/active-language'

const STORAGE_KEY = 'language'
const DEFAULT_LANGUAGE: Language = 'zh'

interface LanguageContextType {
  language: Language
  setLanguage: (lang: Language) => void
}

const LanguageContext = createContext<LanguageContextType | undefined>(
  undefined
)

function isLanguage(value: unknown): value is Language {
  return value === 'en' || value === 'zh'
}

/**
 * Resolve the initial language. An explicit choice made in a previous visit
 * always wins. First-time visitors get Chinese: it is the product's primary
 * market, and the switcher is one click away for everyone else — so we do not
 * try to out-guess the browser's Accept-Language header.
 */
function resolveInitialLanguage(): Language {
  if (typeof window === 'undefined') {
    return DEFAULT_LANGUAGE
  }

  try {
    const saved = window.localStorage.getItem(STORAGE_KEY)
    if (isLanguage(saved)) {
      return saved
    }
  } catch {
    // Private mode / storage disabled — fall through to the default.
  }

  return DEFAULT_LANGUAGE
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>(() => {
    const initial = resolveInitialLanguage()
    // Keep ambient (non-React) translations in sync from the very first render.
    setActiveLanguage(initial)
    return initial
  })

  // Side effects stay out of the state updater: React may invoke an updater
  // more than once (StrictMode), which would double-write storage and the
  // ambient language. Persistence and <html lang> are handled by the effect.
  const setLanguage = useCallback((lang: Language) => {
    setActiveLanguage(lang)
    setLanguageState((current) => (current === lang ? current : lang))
  }, [])

  // Keep <html lang> in sync so screen readers and CSS :lang() rules follow
  // the active language instead of the static value in index.html.
  useEffect(() => {
    const htmlLang = language === 'zh' ? 'zh-CN' : 'en'
    document.documentElement.setAttribute('lang', htmlLang)
    try {
      window.localStorage.setItem(STORAGE_KEY, language)
    } catch {
      // ignore
    }
  }, [language])

  const value = useMemo(
    () => ({ language, setLanguage }),
    [language, setLanguage]
  )

  return (
    <LanguageContext.Provider value={value}>
      {children}
    </LanguageContext.Provider>
  )
}

export function useLanguage() {
  const context = useContext(LanguageContext)
  if (!context) {
    throw new Error('useLanguage must be used within LanguageProvider')
  }
  return context
}
