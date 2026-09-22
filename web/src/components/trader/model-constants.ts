import { tg } from '../../i18n/translations'

// Constants for AI model and provider configuration

export interface AIProviderConfig {
  defaultModel: string
  apiUrl: string
  apiName: string
}

// Get friendly AI model display name
export function getModelDisplayName(modelId: string): string {
  switch (modelId.toLowerCase()) {
    case 'deepseek':
      return 'DeepSeek'
    case 'qwen':
      return 'Qwen'
    case 'claude':
      return 'Claude'
    default:
      return modelId.toUpperCase()
  }
}

// Extract name part after underscore
export function getShortName(fullName: string): string {
  const parts = fullName.split('_')
  return parts.length > 1 ? parts[parts.length - 1] : fullName
}

// Native provider configuration — default model, API key console and display
// name. Must stay in sync with `supportedProviderDefaults` in
// api/handler_ai_model.go and the native clients in mcp/provider/.
export const AI_PROVIDER_CONFIG: Record<string, AIProviderConfig> = {
  deepseek: {
    defaultModel: 'deepseek-chat',
    apiUrl: 'https://platform.deepseek.com/api_keys',
    apiName: 'DeepSeek',
  },
  openai: {
    defaultModel: 'gpt-4o',
    apiUrl: 'https://platform.openai.com/api-keys',
    apiName: 'OpenAI',
  },
  claude: {
    defaultModel: 'claude-sonnet-4-20250514',
    apiUrl: 'https://console.anthropic.com/settings/keys',
    apiName: 'Claude',
  },
  qwen: {
    defaultModel: 'qwen3-max',
    apiUrl: 'https://bailian.console.aliyun.com/',
    apiName: 'Qwen',
  },
  gemini: {
    defaultModel: 'gemini-2.5-pro',
    apiUrl: 'https://aistudio.google.com/app/apikey',
    apiName: 'Gemini',
  },
  grok: {
    defaultModel: 'grok-4',
    apiUrl: 'https://console.x.ai/',
    apiName: 'Grok',
  },
  kimi: {
    defaultModel: 'kimi-k2-0905-preview',
    apiUrl: 'https://platform.moonshot.cn/console/api-keys',
    apiName: 'Kimi',
  },
  minimax: {
    defaultModel: 'MiniMax-M2.7',
    apiUrl: 'https://platform.minimaxi.com/user-center/basic-information/interface-key',
    apiName: 'MiniMax',
  },
}

// Helper function to get exchange display name from exchange ID (UUID)
export function getExchangeDisplayName(
  exchangeId: string | undefined,
  exchanges: {
    id: string
    exchange_type?: string
    name: string
    account_name?: string
  }[]
): string {
  if (!exchangeId) return tg('common.unknown')
  const exchange = exchanges.find((e) => e.id === exchangeId)
  if (!exchange) return exchangeId.substring(0, 8).toUpperCase() + '...' // Show truncated UUID if not found
  const typeName = exchange.exchange_type?.toUpperCase() || exchange.name
  return exchange.account_name
    ? `${typeName} - ${exchange.account_name}`
    : typeName
}

// Helper function to check if exchange is a perp-dex type (wallet-based)
export function isPerpDexExchange(exchangeType: string | undefined): boolean {
  if (!exchangeType) return false
  const perpDexTypes = ['hyperliquid', 'lighter', 'aster']
  return perpDexTypes.includes(exchangeType.toLowerCase())
}

// Helper function to get wallet address for perp-dex exchanges
export function getWalletAddress(
  exchange:
    | {
        exchange_type?: string
        hyperliquidWalletAddr?: string
        lighterWalletAddr?: string
        asterSigner?: string
      }
    | undefined
): string | undefined {
  if (!exchange) return undefined
  const type = exchange.exchange_type?.toLowerCase()
  switch (type) {
    case 'hyperliquid':
      return exchange.hyperliquidWalletAddr
    case 'lighter':
      return exchange.lighterWalletAddr
    case 'aster':
      return exchange.asterSigner
    default:
      return undefined
  }
}

// Helper function to truncate wallet address for display
export function truncateAddress(
  address: string,
  startLen = 6,
  endLen = 4
): string {
  if (address.length <= startLen + endLen + 3) return address
  return `${address.slice(0, startLen)}...${address.slice(-endLen)}`
}
