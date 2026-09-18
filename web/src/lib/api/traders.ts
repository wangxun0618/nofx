import type {
  TraderInfo,
  TraderConfigData,
  CreateTraderRequest,
} from '../../types'
import { API_BASE, httpClient } from './helpers'
import { ApiError } from '../httpClient'

import { tg } from '../../i18n/translations'

// Create/update/start legitimately run long: stopping a live trader waits for
// its in-flight cycle and monitors, and creation probes the exchange (~35s
// worst case observed). The default 30s axios timeout aborts mid-operation and
// reports a false failure, so these calls get their own generous ceiling.
const TRADER_LIFECYCLE_TIMEOUT_MS = 120_000

function throwApiError(
  message: string,
  errorKey?: string,
  errorParams?: Record<string, string>,
  statusCode?: number,
  errorData?: Record<string, any>
): never {
  throw new ApiError(message, errorKey, errorParams, statusCode, errorData)
}

export const traderApi = {
  async getTraders(silent?: boolean): Promise<TraderInfo[]> {
    const result = await httpClient.request<TraderInfo[]>(
      `${API_BASE}/my-traders`,
      { silent }
    )
    if (!result.success) throw new Error(tg('lib.fetchTraderList'))
    return Array.isArray(result.data) ? result.data : []
  },

  async getPublicTraders(): Promise<any[]> {
    const result = await httpClient.get<any[]>(`${API_BASE}/traders`)
    if (!result.success) throw new Error(tg('lib.fetchPublicTraderList'))
    return result.data!
  },

  async createTrader(request: CreateTraderRequest): Promise<TraderInfo> {
    const result = await httpClient.request<TraderInfo>(`${API_BASE}/traders`, {
      method: 'POST',
      data: request,
      timeout: TRADER_LIFECYCLE_TIMEOUT_MS,
    })
    if (!result.success) {
      throwApiError(
        result.message || tg('lib.createTrader'),
        result.errorKey,
        result.errorParams,
        result.statusCode
      )
    }
    return result.data!
  },

  async deleteTrader(traderId: string): Promise<void> {
    const result = await httpClient.delete(`${API_BASE}/traders/${traderId}`)
    if (!result.success) throw new Error(tg('lib.deleteTrader'))
  },

  async startTrader(traderId: string): Promise<void> {
    const result = await httpClient.request(
      `${API_BASE}/traders/${traderId}/start`,
      { method: 'POST', timeout: TRADER_LIFECYCLE_TIMEOUT_MS }
    )
    if (!result.success) {
      throwApiError(
        result.message || tg('lib.startTrader'),
        result.errorKey,
        result.errorParams,
        result.statusCode,
        result.errorData
      )
    }
  },

  async stopTrader(traderId: string): Promise<void> {
    const result = await httpClient.post(`${API_BASE}/traders/${traderId}/stop`)
    if (!result.success) throw new Error(tg('lib.stopTrader'))
  },

  async toggleCompetition(traderId: string, showInCompetition: boolean): Promise<void> {
    const result = await httpClient.put(
      `${API_BASE}/traders/${traderId}/competition`,
      { show_in_competition: showInCompetition }
    )
    if (!result.success) throw new Error(tg('lib.updateCompetitionVisibility'))
  },

  async closePosition(traderId: string, symbol: string, side: string): Promise<{ message: string }> {
    const result = await httpClient.post<{ message: string }>(
      `${API_BASE}/traders/${traderId}/close-position`,
      { symbol, side }
    )
    if (!result.success) throw new Error(tg('lib.closePosition'))
    return result.data!
  },

  async updateTraderPrompt(
    traderId: string,
    customPrompt: string
  ): Promise<void> {
    const result = await httpClient.put(
      `${API_BASE}/traders/${traderId}/prompt`,
      { custom_prompt: customPrompt }
    )
    if (!result.success) throw new Error(tg('lib.updateCustomPrompt'))
  },

  async getTraderConfig(
    traderId: string,
    silent?: boolean
  ): Promise<TraderConfigData> {
    const result = await httpClient.request<TraderConfigData>(
      `${API_BASE}/traders/${traderId}/config`,
      { silent }
    )
    if (!result.success) throw new Error(tg('lib.fetchTraderConfig'))
    return result.data!
  },

  async updateTrader(
    traderId: string,
    request: CreateTraderRequest
  ): Promise<TraderInfo> {
    const result = await httpClient.request<TraderInfo>(
      `${API_BASE}/traders/${traderId}`,
      { method: 'PUT', data: request, timeout: TRADER_LIFECYCLE_TIMEOUT_MS }
    )
    if (!result.success) {
      throwApiError(
        result.message || tg('lib.updateTrader'),
        result.errorKey,
        result.errorParams,
        result.statusCode
      )
    }
    return result.data!
  },
}
