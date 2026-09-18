import type { TelegramConfig } from '../../types'
import { API_BASE, httpClient } from './helpers'

import { tg } from '../../i18n/translations'

export const telegramApi = {
  async getTelegramConfig(): Promise<TelegramConfig> {
    const result = await httpClient.get<TelegramConfig>(`${API_BASE}/telegram`)
    if (!result.success) throw new Error(tg('lib.fetchTelegramConfig'))
    return result.data!
  },

  async updateTelegramConfig(token: string, modelId?: string): Promise<void> {
    const result = await httpClient.post(`${API_BASE}/telegram`, { bot_token: token, model_id: modelId ?? '' })
    if (!result.success) throw new Error(tg('lib.saveTelegramConfig'))
  },

  async unbindTelegram(): Promise<void> {
    const result = await httpClient.delete(`${API_BASE}/telegram/binding`)
    if (!result.success) throw new Error(tg('lib.unbindTelegram'))
  },

  async updateTelegramModel(modelId: string): Promise<void> {
    const result = await httpClient.post(`${API_BASE}/telegram/model`, { model_id: modelId })
    if (!result.success) throw new Error(tg('lib.updateTelegramModel'))
  },
}
