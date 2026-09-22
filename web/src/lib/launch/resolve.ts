import { api } from '../api'
import type { AIModel, Exchange } from '../../types'

import { tg } from '../../i18n/translations'

export function modelHasCredential(model: AIModel) {
  return Boolean(model.has_api_key || model.apiKey)
}

export function exchangeHasKey(exchange: Exchange) {
  return Boolean(exchange.has_api_key || exchange.apiKey)
}

export function isHyperliquidExchange(exchange: Exchange) {
  return exchange.exchange_type === 'hyperliquid'
}

/** The first enabled model that has a stored API key. */
export function pickTradingModel(models: AIModel[]) {
  return models.find((model) => model.enabled && modelHasCredential(model)) || null
}

export function pickTradingExchange(exchanges: Exchange[]) {
  return (
    exchanges.find(
      (exchange) =>
        isHyperliquidExchange(exchange) &&
        exchange.enabled &&
        exchangeHasKey(exchange) &&
        Boolean(exchange.hyperliquidBuilderApproved) &&
        (exchange.hyperliquidWalletAddr || '').trim() !== ''
    ) || null
  )
}

/**
 * Resolves a launch-capable AI model. Returns null when none is configured —
 * the caller routes the user into AI-model setup.
 */
export async function resolveLaunchModel(): Promise<AIModel | null> {
  const models = await api.getModelConfigs()
  return pickTradingModel(models)
}

/**
 * Resolves a launch-capable exchange. Returns the exchange or a message
 * explaining the most specific missing prerequisite.
 */
export async function resolveLaunchExchange(): Promise<
  { exchange: Exchange } | { exchange: null; reason: string }
> {
  const exchanges = await api.getExchangeConfigs()
  const ready = pickTradingExchange(exchanges)
  if (ready) return { exchange: ready }

  const hyperliquid = exchanges.find(isHyperliquidExchange)
  if (!hyperliquid) {
    return {
      exchange: null,
      reason:
        tg('lib.noHyperliquidAccount'),
    }
  }
  if (!hyperliquid.enabled) {
    return {
      exchange: null,
      reason: tg('lib.hyperliquidDisabled'),
    }
  }
  if (!exchangeHasKey(hyperliquid)) {
    return {
      exchange: null,
      reason:
        tg('lib.hyperliquidAgentKeyMissing'),
    }
  }
  if (!hyperliquid.hyperliquidBuilderApproved) {
    return {
      exchange: null,
      reason:
        tg('lib.hyperliquidBuilderIncomplete'),
    }
  }
  return {
    exchange: null,
    reason:
      tg('lib.hyperliquidAddressMissing'),
  }
}
