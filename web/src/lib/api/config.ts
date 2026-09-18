import type {
  AIModel,
  Exchange,
  ExchangeAccountStateResponse,
  UpdateModelConfigRequest,
  UpdateExchangeConfigRequest,
  CreateExchangeRequest,
  BeginnerOnboardingResponse,
  CurrentBeginnerWalletResponse,
} from '../../types'
import { API_BASE, httpClient, CryptoService } from './helpers'

import { tg } from '../../i18n/translations'

export const configApi = {
  async getModelConfigs(): Promise<AIModel[]> {
    const result = await httpClient.get<AIModel[]>(`${API_BASE}/models`)
    if (!result.success) throw new Error(tg('lib.fetchModelConfigs'))
    return Array.isArray(result.data) ? result.data : []
  },

  async getSupportedModels(): Promise<AIModel[]> {
    const result = await httpClient.get<AIModel[]>(
      `${API_BASE}/supported-models`
    )
    if (!result.success) throw new Error(tg('lib.fetchSupportedModels'))
    return result.data!
  },

  async getPromptTemplates(): Promise<string[]> {
    const res = await fetch(`${API_BASE}/prompt-templates`)
    if (!res.ok) throw new Error(tg('lib.fetchPromptTemplates'))
    const data = await res.json()
    if (Array.isArray(data.templates)) {
      return data.templates.map((item: { name: string }) => item.name)
    }
    return []
  },

  async updateModelConfigs(request: UpdateModelConfigRequest): Promise<void> {
    // Check if transport encryption is enabled
    const config = await CryptoService.fetchCryptoConfig()

    if (!config.transport_encryption) {
      // Transport encryption disabled, send plaintext
      const result = await httpClient.put(`${API_BASE}/models`, request)
      if (!result.success) throw new Error(tg('lib.updateModelConfigs'))
      return
    }

    // Fetch RSA public key
    const publicKey = await CryptoService.fetchPublicKey()

    // Initialize crypto service
    await CryptoService.initialize(publicKey)

    // Get user info from localStorage
    const userId = localStorage.getItem('user_id') || ''
    const sessionId = sessionStorage.getItem('session_id') || ''

    // Encrypt sensitive data
    const encryptedPayload = await CryptoService.encryptSensitiveData(
      JSON.stringify(request),
      userId,
      sessionId
    )

    // Send encrypted data
    const result = await httpClient.put(`${API_BASE}/models`, encryptedPayload)
    if (!result.success) throw new Error(tg('lib.updateModelConfigs'))
  },

  async getExchangeConfigs(): Promise<Exchange[]> {
    const result = await httpClient.get<Exchange[]>(`${API_BASE}/exchanges`)
    if (!result.success) throw new Error(tg('lib.fetchExchangeConfigs'))
    return result.data!
  },

  async getExchangeAccountState(): Promise<ExchangeAccountStateResponse> {
    const result = await httpClient.get<ExchangeAccountStateResponse>(
      `${API_BASE}/exchanges/account-state`
    )
    if (!result.success || !result.data) {
      throw new Error(tg('lib.fetchExchangeAccountStates'))
    }
    return result.data
  },

  async getSupportedExchanges(): Promise<Exchange[]> {
    const result = await httpClient.get<Exchange[]>(
      `${API_BASE}/supported-exchanges`
    )
    if (!result.success) throw new Error(tg('lib.fetchSupportedExchanges'))
    return result.data!
  },

  async updateExchangeConfigs(
    request: UpdateExchangeConfigRequest
  ): Promise<void> {
    const result = await httpClient.put(`${API_BASE}/exchanges`, request)
    if (!result.success) throw new Error(tg('lib.updateExchangeConfigs'))
  },

  async createExchange(request: CreateExchangeRequest): Promise<{ id: string }> {
    const result = await httpClient.post<{ id: string }>(`${API_BASE}/exchanges`, request)
    if (!result.success) throw new Error(tg('lib.createExchangeAccount'))
    return result.data!
  },

  async createExchangeEncrypted(request: CreateExchangeRequest): Promise<{ id: string }> {
    // Check if transport encryption is enabled
    const config = await CryptoService.fetchCryptoConfig()

    if (!config.transport_encryption) {
      // Transport encryption disabled, send plaintext
      const result = await httpClient.post<{ id: string }>(`${API_BASE}/exchanges`, request)
      if (!result.success) throw new Error(tg('lib.createExchangeAccount'))
      return result.data!
    }

    // Fetch RSA public key
    const publicKey = await CryptoService.fetchPublicKey()

    // Initialize crypto service
    await CryptoService.initialize(publicKey)

    // Get user info
    const userId = localStorage.getItem('user_id') || ''
    const sessionId = sessionStorage.getItem('session_id') || ''

    // Encrypt sensitive data
    const encryptedPayload = await CryptoService.encryptSensitiveData(
      JSON.stringify(request),
      userId,
      sessionId
    )

    // Send encrypted data
    const result = await httpClient.post<{ id: string }>(
      `${API_BASE}/exchanges`,
      encryptedPayload
    )
    if (!result.success) throw new Error(tg('lib.createExchangeAccount'))
    return result.data!
  },

  async deleteExchange(exchangeId: string): Promise<void> {
    const result = await httpClient.delete(`${API_BASE}/exchanges/${exchangeId}`)
    if (!result.success) throw new Error(tg('lib.deleteExchangeAccount'))
  },

  async updateExchangeConfigsEncrypted(
    request: UpdateExchangeConfigRequest
  ): Promise<void> {
    // Check if transport encryption is enabled
    const config = await CryptoService.fetchCryptoConfig()

    if (!config.transport_encryption) {
      // Transport encryption disabled, send plaintext
      const result = await httpClient.put(`${API_BASE}/exchanges`, request)
      if (!result.success) throw new Error(tg('lib.updateExchangeConfigs'))
      return
    }

    // Fetch RSA public key
    const publicKey = await CryptoService.fetchPublicKey()

    // Initialize crypto service
    await CryptoService.initialize(publicKey)

    // Get user info from localStorage
    const userId = localStorage.getItem('user_id') || ''
    const sessionId = sessionStorage.getItem('session_id') || ''

    // Encrypt sensitive data
    const encryptedPayload = await CryptoService.encryptSensitiveData(
      JSON.stringify(request),
      userId,
      sessionId
    )

    // Send encrypted data
    const result = await httpClient.put(
      `${API_BASE}/exchanges`,
      encryptedPayload
    )
    if (!result.success) throw new Error(tg('lib.updateExchangeConfigs'))
  },

  async getServerIP(): Promise<{
    public_ip: string
    message: string
  }> {
    const result = await httpClient.get<{
      public_ip: string
      message: string
    }>(`${API_BASE}/server-ip`)
    if (!result.success) throw new Error(tg('lib.fetchServerIp'))
    return result.data!
  },

  async prepareBeginnerOnboarding(): Promise<BeginnerOnboardingResponse> {
    const result = await httpClient.post<BeginnerOnboardingResponse>(
      `${API_BASE}/onboarding/beginner`
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || tg('lib.prepareBeginnerOnboarding'))
    }
    return result.data
  },

  async getCurrentBeginnerWallet(): Promise<CurrentBeginnerWalletResponse> {
    const result = await httpClient.get<CurrentBeginnerWalletResponse>(
      `${API_BASE}/onboarding/beginner/current`
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || tg('lib.fetchBeginnerWallet'))
    }
    return result.data
  },
}
