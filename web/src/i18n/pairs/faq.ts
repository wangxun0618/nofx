import type { PairMap } from '../translation-pair'

// FAQ page UI chrome — article prose lives in data/faqData.ts as a parallel
// localized array, not here. Both languages are required for every key.
export const faqPairs: PairMap = {
  'faq.pageTitle': { en: 'FAQ', zh: '常见问题' },
  'faq.subtitle': {
    en: '{count} answers · wallets · launch · trading · self-hosting',
    zh: '{count} 个解答 · 钱包 · 启动 · 交易 · 自托管',
  },
  'faq.searchPlaceholder': { en: 'Search FAQ...', zh: '搜索常见问题…' },
  'faq.noResults': {
    en: 'No matching questions for “{term}”.',
    zh: '没有与“{term}”匹配的问题。',
  },
  'faq.clearSearch': { en: 'Clear search', zh: '清除搜索' },
  'faq.stillHaveQuestions': { en: 'Still have questions?', zh: '还有疑问？' },
  'faq.stillHaveQuestionsDesc': {
    en: 'Ask in the community or open an issue — both are answered by the people building NOFX.',
    zh: '在社群中提问或提交 issue —— 都由 NOFX 的开发团队亲自解答。',
  },
  'faq.telegramCommunity': { en: 'Telegram community', zh: 'Telegram 社区' },
  'faq.entry': { en: 'entry', zh: '条' },
  'faq.entries': { en: 'entries', zh: '条' },
}
