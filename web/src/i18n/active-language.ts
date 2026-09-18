import type { Language } from './translations'

/**
 * Ambient language for code that cannot use the React context — data modules,
 * API clients and other non-component files. `LanguageProvider` keeps this in
 * sync with the active language so plain helper functions can still translate.
 *
 * Prefer `t(key, language)` / `useLanguage()` inside components; reach for
 * `tg(key)` only outside the React tree.
 */
let activeLanguage: Language = 'zh'

export function setActiveLanguage(language: Language) {
  activeLanguage = language
}

export function getActiveLanguage(): Language {
  return activeLanguage
}
