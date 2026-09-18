/**
 * A single piece of UI copy in both supported languages. Declaring both
 * languages on the same line makes translation parity structural — a key
 * cannot exist for one language only.
 */
export interface TranslationPair {
  en: string
  zh: string
}

/**
 * A flat map of dotted key -> pair, e.g.
 *   { 'faq.title': { en: 'FAQ', zh: '常见问题' } }
 * The dotted prefix becomes a namespace inside the merged dictionary.
 */
export type PairMap = Record<string, TranslationPair>
