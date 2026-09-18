import {
  BookOpen,
  GitBranch,
  Monitor,
  Shield,
  TrendingUp,
  Wrench,
  Zap,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { Language } from '../i18n/translations'

/**
 * FAQ content model. Answers are composed from typed blocks so the renderer
 * stays generic — no per-question JSX special cases. Inline `code` spans are
 * written with backticks and parsed by the renderer.
 */
export type FAQBlock =
  | { type: 'p'; text: string }
  | { type: 'list'; items: string[] }
  | { type: 'steps'; items: string[] }
  | { type: 'note'; text: string }
  | { type: 'links'; links: { label: string; href: string }[] }

export interface FAQItem {
  id: string
  question: string
  blocks: FAQBlock[]
}

export interface FAQCategory {
  id: string
  title: string
  icon: LucideIcon
  items: FAQItem[]
}

/** Plain text of an item, used by the search filter. */
export function faqItemSearchText(item: FAQItem): string {
  const parts: string[] = [item.question]
  for (const block of item.blocks) {
    if (block.type === 'p' || block.type === 'note') parts.push(block.text)
    else if (block.type === 'list' || block.type === 'steps')
      parts.push(block.items.join(' '))
    else if (block.type === 'links')
      parts.push(block.links.map((l) => l.label).join(' '))
  }
  return parts.join(' ').toLowerCase()
}

// ───────────────────────── English source ─────────────────────────
export const faqCategoriesEn: FAQCategory[] = [
  // ───────────────────────── Getting started ─────────────────────────
  {
    id: 'getting-started',
    title: 'Getting Started',
    icon: BookOpen,
    items: [
      {
        id: 'what-is-nofx',
        question: 'What is NOFX?',
        blocks: [
          {
            type: 'p',
            text: 'NOFX is an open-source, self-hosted AI trading terminal. Its flagship mode is the NOFX Autopilot: an AI agent that reads the Claw402.ai signal board, verifies candidates with Signal Lab and liquidation structure, confirms timing with raw candles, and executes on Hyperliquid — all on your own machine, with your keys never leaving your server.',
          },
          {
            type: 'p',
            text: 'Beyond the Autopilot you can build custom strategies in Strategy Studio, run multiple AI traders side by side, and compare them on the leaderboard.',
          },
        ],
      },
      {
        id: 'what-do-i-need',
        question: 'What do I need before launching the Autopilot?',
        blocks: [
          {
            type: 'p',
            text: 'Two funded accounts — the guided launch on the Config page walks you through both:',
          },
          {
            type: 'list',
            items: [
              'An AI fee wallet: a Base-chain USDC wallet that pays for AI model and market-data calls. Minimum `1 USDC` to launch.',
              'A Hyperliquid account with trading authorization and at least `12 USDC` available as margin.',
            ],
          },
          {
            type: 'p',
            text: 'The launch button runs a server-side preflight that checks every prerequisite and points you at the exact step that is missing, so you cannot start a half-configured bot.',
          },
        ],
      },
      {
        id: 'which-markets',
        question: 'Which markets can it trade?',
        blocks: [
          {
            type: 'p',
            text: 'The Autopilot trades Hyperliquid perpetuals: crypto majors (BTC, ETH, SOL, …) plus the xyz synthetic markets covering US stocks, indices, commodities, and FX — so one account gives the AI a multi-asset universe.',
          },
          {
            type: 'p',
            text: 'Manual traders built in Strategy Studio can also connect Binance, Bybit, OKX, Bitget, KuCoin, Gate, Aster, and Lighter.',
          },
        ],
      },
      {
        id: 'ai-models',
        question: 'Which AI models does it use? Do I need API keys?',
        blocks: [
          {
            type: 'p',
            text: 'No API keys are required. NOFX routes inference through Claw402 pay-as-you-go infrastructure: your AI fee wallet pays per call with Base USDC, and the terminal accesses supported models (DeepSeek and other frontier models) on demand.',
          },
          {
            type: 'p',
            text: 'Power users can still plug in their own provider keys (OpenAI, Claude, Gemini, DeepSeek, Qwen, Grok, Kimi, or any OpenAI-compatible endpoint) under Config → Models.',
          },
        ],
      },
      {
        id: 'is-it-profitable',
        question: 'Will it make money?',
        blocks: [
          {
            type: 'p',
            text: 'No one can promise that, and you should distrust anyone who does. The AI trades a systematic process, but markets are adversarial and past performance never guarantees future results.',
          },
          {
            type: 'p',
            text: 'The dashboard is deliberately honest about performance: it separates realized from unrealized P/L, shows the fee-drag chain (gross − fees = net), profit factor, and max drawdown computed from your real starting balance. Watch those numbers, start small, and only trade money you can afford to lose.',
          },
          {
            type: 'note',
            text: 'Trading involves substantial risk of loss. NOFX is software, not investment advice.',
          },
        ],
      },
    ],
  },

  // ───────────────────────── Launch & wallets ─────────────────────────
  {
    id: 'launch-wallets',
    title: 'Launch & Wallets',
    icon: Zap,
    items: [
      {
        id: 'ai-fee-wallet',
        question: 'What is the AI fee wallet?',
        blocks: [
          {
            type: 'p',
            text: 'A dedicated EVM wallet on Base that pays for AI model calls and paid market data (x402 micropayments). It is completely separate from your trading collateral — it never touches Hyperliquid.',
          },
          {
            type: 'list',
            items: [
              'The guided setup creates it for you (or reuses an existing one).',
              'Deposit only USDC on the Base network to its address.',
              'Launch requires at least `1 USDC`; the balance display refreshes automatically after a deposit.',
              'A typical cycle costs a fraction of a cent to a few cents depending on the model.',
            ],
          },
        ],
      },
      {
        id: 'fee-wallet-private-key',
        question: 'Where is the AI fee wallet private key kept?',
        blocks: [
          {
            type: 'p',
            text: 'The key is generated locally on your server, stored encrypted (AES-256) in your own database, and shown to you once in the onboarding screen. Back it up — it cannot be recovered if you lose your database.',
          },
          {
            type: 'note',
            text: 'Keep only fee money in this wallet. It exists to pay for AI calls, not to hold savings.',
          },
        ],
      },
      {
        id: 'hyperliquid-authorization',
        question: 'How does the Hyperliquid authorization work? Is it safe?',
        blocks: [
          {
            type: 'p',
            text: 'NOFX uses Hyperliquid agent wallets, so your main wallet key is never shared. The connect flow has four signed steps:',
          },
          {
            type: 'steps',
            items: [
              'Connect your EVM wallet (Rabby, MetaMask, OKX, Coinbase Wallet).',
              'Approve a freshly generated NOFX agent wallet — valid for 180 days, trading only.',
              'Approve the builder fee (a small per-order fee that funds the platform).',
              'Save the agent key to your NOFX server (stored encrypted).',
            ],
          },
          {
            type: 'p',
            text: 'The agent wallet can place and close orders, nothing else. It cannot withdraw funds, and your collateral stays inside your own Hyperliquid account at all times.',
          },
        ],
      },
      {
        id: 'launch-preflight',
        question: 'What does the launch preflight check?',
        blocks: [
          {
            type: 'p',
            text: 'Before anything is created or changed, the server verifies the full chain with live data:',
          },
          {
            type: 'list',
            items: [
              'AI model is enabled and has a credential.',
              'AI fee wallet key is valid and the Base USDC balance is at least `1 USDC` (queried on-chain).',
              'Hyperliquid account is authorized (agent + builder fee) and reachable.',
              'Trading funds: at least `12 USDC` counting equity in open positions.',
            ],
          },
          {
            type: 'p',
            text: 'Each failing check names the exact fix and deep-links into the guided setup. The same checks are enforced server-side on every start, so the UI cannot be bypassed accidentally.',
          },
        ],
      },
      {
        id: 'relaunch-behavior',
        question: 'What happens if I press Launch again?',
        blocks: [
          {
            type: 'p',
            text: 'Launching is idempotent. If a NOFX Autopilot already exists, the launcher updates it with the current strategy config and restarts it — it never creates a duplicate. A restart can take up to a minute if the bot is mid-cycle; the UI waits for it.',
          },
        ],
      },
      {
        id: 'deposit-not-showing',
        question: 'I deposited USDC but the balance still shows zero.',
        blocks: [
          {
            type: 'list',
            items: [
              'AI fee wallet: make sure you sent USDC on the Base network to the exact address shown. Balances are cached for ~30 seconds, and the setup panel re-checks automatically every few seconds.',
              'Hyperliquid: deposits land in your own Hyperliquid account; the balance step polls the live account state. Use Refresh in the guided panel if in doubt.',
              'If the on-chain RPC is temporarily unreachable, the panel marks the balance as unknown instead of zero — retry in a minute.',
            ],
          },
        ],
      },
    ],
  },

  // ───────────────────────── Trading & execution ─────────────────────────
  {
    id: 'trading',
    title: 'Trading & Execution',
    icon: TrendingUp,
    items: [
      {
        id: 'autopilot-pipeline',
        question: 'How does the Autopilot strategy work, step by step?',
        blocks: [
          {
            type: 'p',
            text: 'Every cycle runs the same four-stage funnel — each stage can reject a candidate, and only setups that survive all four get traded:',
          },
          {
            type: 'steps',
            items: [
              'Build the universe — pull the live Claw402.ai ranking and take the top candidates (default 10) across crypto majors and the xyz synthetic markets (US stocks, indices, commodities, FX), each with a direction bias and signal z-score.',
              'Verify each candidate — fetch its Signal Lab deep signal plus the cost/liquidation structure around price: liquidation clusters and cost bases show whether a move has fuel ahead of it or walls against it.',
              'Confirm timing — read the raw 15-minute OHLCV candles (30 bars) to check the entry is riding structure, not chasing an extended move.',
              'Decide and size — only setups clearing the confidence threshold (default `78/100`) with roughly `3:1` risk/reward get positions at 10x; closes always execute before opens, and both long and short books are considered every cycle.',
            ],
          },
          {
            type: 'p',
            text: 'A fifth layer sits outside the AI entirely: hard risk controls (position cap, leverage caps, margin ceiling, trade throttle) reject any decision that violates them, no matter how confident the model is.',
          },
        ],
      },
      {
        id: 'data-sources',
        question: 'What data does it use, and which parts are paid?',
        blocks: [
          {
            type: 'list',
            items: [
              'Claw402.ai signal data — the ranking board, per-symbol Signal Lab deep signals, and market net-flow. These are paid endpoints, billed per call in USDC from your AI fee wallet (x402 micropayments).',
              'Cost/liquidation heatmaps — aggregated position-cost and liquidation-cluster structure for each market.',
              'Hyperliquid market data — raw OHLCV candles and the live L2 order book (free public data).',
              'Your account via the agent wallet — equity, available margin, open positions with PnL.',
              'Its own history — closed trades feed win rate, profit factor and drawdown back into the next prompt, so the AI knows its recent form.',
            ],
          },
          {
            type: 'note',
            text: 'The dashboard polls the paid Claw402 endpoints slowly (every few minutes) to conserve your fee wallet — market-data panels stay live from the free feeds.',
          },
        ],
      },
      {
        id: 'decision-cycle',
        question: 'How often does the AI make decisions?',
        blocks: [
          {
            type: 'p',
            text: 'The Autopilot runs a scan cycle every 5–15 minutes depending on how you launched it (configurable per trader, minimum 3 minutes). The first cycle starts right after launch; a single cycle usually takes 30–60 seconds because the AI reads the full market context before deciding.',
          },
        ],
      },
      {
        id: 'what-ai-sees',
        question: 'What information does the AI see each cycle?',
        blocks: [
          {
            type: 'list',
            items: [
              'Your account: equity, available margin, open positions with PnL.',
              'The Claw402 ranking board: candidate universe with direction bias.',
              'Signal Lab deep signals and cost/liquidation structure per candidate.',
              'Raw OHLCV candles for timing confirmation.',
              'Its own track record: win rate, profit factor, drawdown, recent trades.',
            ],
          },
          {
            type: 'p',
            text: 'Every cycle is stored as a decision record — the Execution Log on the dashboard shows the reasoning chain, actions, and any blocked orders.',
          },
        ],
      },
      {
        id: 'leverage-and-risk',
        question: 'What leverage and risk controls does it use?',
        blocks: [
          {
            type: 'p',
            text: 'The Autopilot defaults to 10x cross margin. Hard risk controls run outside the AI and cannot be overridden by it:',
          },
          {
            type: 'list',
            items: [
              'Position-count cap from the strategy config — new opens are rejected at the cap.',
              'Leverage limits per asset class (BTC/ETH vs altcoins).',
              'A trade throttle blocks churn, e.g. closing a barely-moved position minutes after opening it.',
              'Safe mode (below) protects the book when the AI itself is failing.',
            ],
          },
        ],
      },
      {
        id: 'safe-mode',
        question: 'What is safe mode?',
        blocks: [
          {
            type: 'p',
            text: 'If the AI fails 3 cycles in a row (provider outage, empty fee wallet, bad responses), the trader enters safe mode: no new positions are opened, existing positions keep their protection, and the loop keeps retrying. It exits safe mode automatically on the next successful AI call.',
          },
          {
            type: 'p',
            text: 'Safe mode is shown as a banner on the dashboard together with the reason, so it never happens silently.',
          },
        ],
      },
      {
        id: 'fee-wallet-empty-mid-run',
        question: 'What happens if the AI fee wallet runs out mid-run?',
        blocks: [
          {
            type: 'p',
            text: 'AI calls start failing with a clear "out of funds" status. The dashboard shows a persistent red banner with the wallet balance, and after three failed cycles the bot enters safe mode. Top up Base USDC to the fee wallet and the trader recovers on its own — no restart needed.',
          },
        ],
      },
      {
        id: 'trading-fees',
        question: 'What fees am I paying?',
        blocks: [
          {
            type: 'list',
            items: [
              'Hyperliquid trading fees on every order, plus the approved builder fee.',
              'AI/data costs paid per call from the fee wallet (cents per cycle).',
            ],
          },
          {
            type: 'p',
            text: 'Fees are the silent killer of high-frequency strategies. The dashboard stats strip shows the full chain — gross realized P/L, minus fees, equals net — so you can see immediately whether fees are eating the edge.',
          },
        ],
      },
      {
        id: 'stop-and-manual',
        question: 'How do I stop the bot or close positions manually?',
        blocks: [
          {
            type: 'list',
            items: [
              'Stop: use the Stop button on the Config page trader list. Stopping halts the decision loop; open positions remain open and are yours to manage.',
              'Manual close: close any position from the dashboard positions panel — manual closes sync back into the position history.',
              'Emergencies: you can always manage positions directly on Hyperliquid; NOFX never locks you out of your own account.',
            ],
          },
        ],
      },
    ],
  },

  // ───────────────────────── Dashboard & metrics ─────────────────────────
  {
    id: 'dashboard',
    title: 'Dashboard & Metrics',
    icon: Monitor,
    items: [
      {
        id: 'metrics-meaning',
        question: 'What do the header metrics mean exactly?',
        blocks: [
          {
            type: 'list',
            items: [
              'Equity — live account value including unrealized PnL.',
              'Total P/L (incl. unrealized) — equity versus your starting balance; moves with open positions.',
              'Realized P/L (closed trades) — net result of finished trades only; this is what win rate, profit factor and sharpe are computed from.',
              'Profit factor — gross wins ÷ gross losses on closed trades; above 1.0 means the closed book is net positive.',
              'Max drawdown — worst peak-to-trough dip of the realized equity curve, measured against your real starting balance.',
            ],
          },
        ],
      },
      {
        id: 'pl-contradiction',
        question: 'Why is Total P/L positive while Realized P/L is negative?',
        blocks: [
          {
            type: 'p',
            text: 'They measure different things. Realized P/L only counts closed trades; Total P/L also includes the unrealized gains of positions still open. A bot can be down on its closed trades while its open book carries enough unrealized profit to put total P/L in the green — and vice versa. Check the gross/fees/net strip to see how much of the realized result is fee drag.',
          },
        ],
      },
      {
        id: 'execution-log',
        question: 'Where can I see why the AI did (or refused) something?',
        blocks: [
          {
            type: 'p',
            text: 'The Execution Log panel lists every cycle with the actions taken, the AI call duration, and any blocked orders with the exact guard that fired (throttle, position cap, risk control). Full reasoning chains are stored with each decision record.',
          },
        ],
      },
      {
        id: 'competition',
        question: 'What is the leaderboard / competition?',
        blocks: [
          {
            type: 'p',
            text: 'Traders with "show in competition" enabled appear on the public leaderboard, ranked by live performance. It is opt-in per trader and can be toggled from the trader list at any time.',
          },
        ],
      },
    ],
  },

  // ───────────────────────── Security ─────────────────────────
  {
    id: 'security',
    title: 'Security',
    icon: Shield,
    items: [
      {
        id: 'key-storage',
        question: 'How are my keys stored?',
        blocks: [
          {
            type: 'list',
            items: [
              'All secrets (agent keys, fee wallet key, exchange API keys) are AES-256 encrypted at rest in your own database.',
              'Optional RSA transport encryption protects secrets in flight between browser and server.',
              'NOFX is self-hosted: nothing is sent to any third-party server. The code is open source and auditable.',
            ],
          },
        ],
      },
      {
        id: 'can-nofx-steal-funds',
        question: 'Can NOFX withdraw or steal my funds?',
        blocks: [
          {
            type: 'p',
            text: 'No. On Hyperliquid, NOFX only ever holds an agent wallet, which by protocol design can trade but cannot withdraw. Your collateral stays in your own account under your main wallet’s control.',
          },
          {
            type: 'note',
            text: 'If you connect a CEX instead, create its API key with trading permission only — disable withdrawals and set an IP whitelist.',
          },
        ],
      },
      {
        id: 'registration-model',
        question: 'Why can’t anyone else register on my instance?',
        blocks: [
          {
            type: 'p',
            text: 'By design, an instance is single-operator: the first account registered becomes the operator and registration closes ("System already initialized"). This prevents strangers from creating accounts on an exposed deployment. Run one instance per operator.',
          },
        ],
      },
    ],
  },

  // ───────────────────────── Self-hosting ─────────────────────────
  {
    id: 'self-hosting',
    title: 'Self-Hosting & Troubleshooting',
    icon: Wrench,
    items: [
      {
        id: 'how-to-install',
        question: 'How do I install NOFX?',
        blocks: [
          {
            type: 'p',
            text: 'One line on Linux/macOS (installs and starts everything via Docker):',
          },
          {
            type: 'list',
            items: [
              'Script: `curl -fsSL https://raw.githubusercontent.com/NoFxAiOS/nofx/main/install.sh | bash`',
              'Docker: download `docker-compose.prod.yml` and run `docker compose -f docker-compose.prod.yml up -d`',
              'Windows: install Docker Desktop, then use the Docker route above.',
              'From source: Go 1.21+, Node 18+, TA-Lib (`brew install ta-lib` / `apt-get install libta-lib0-dev`), then `go run .` and `npm --prefix web run dev`.',
            ],
          },
          {
            type: 'p',
            text: 'Then open `http://127.0.0.1:3000` — the web UI is on port 3000, the API on 8080.',
          },
        ],
      },
      {
        id: 'how-to-update',
        question: 'How do I update?',
        blocks: [
          {
            type: 'p',
            text: 'Re-run the install script, or with Docker: `docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml up -d`. Your database and keys live in the mounted `data/` directory and survive updates. Running traders are restarted automatically after the backend comes back.',
          },
        ],
      },
      {
        id: 'launch-blocked',
        question: 'Launch is blocked by a failing check — now what?',
        blocks: [
          {
            type: 'p',
            text: 'Read the message: every preflight failure names its fix and routes you to the right setup step — fund the AI wallet, finish the Hyperliquid authorization, or deposit trading USDC. Balances are re-checked live, so once you fix the item the launch goes through.',
          },
        ],
      },
      {
        id: 'exchange-unreachable',
        question: 'The exchange account shows "invalid credentials" or "unavailable".',
        blocks: [
          {
            type: 'list',
            items: [
              'Invalid credentials: the agent authorization expired (180 days) or the saved key is stale — reconnect the Hyperliquid wallet; the flow offers a one-click renewal.',
              'Unavailable: the exchange API did not respond; the account state is cached for 30 seconds, so wait and refresh.',
              'CEX keys: verify trading permission, IP whitelist, and that futures/perp access is enabled.',
            ],
          },
        ],
      },
      {
        id: 'where-are-logs',
        question: 'Where are the logs?',
        blocks: [
          {
            type: 'list',
            items: [
              'Backend: `docker logs nofx-trading` (or the terminal running `go run .`).',
              'Per-cycle AI reasoning and errors: the dashboard Execution Log.',
              'Frontend build/runtime issues: browser devtools console.',
            ],
          },
        ],
      },
      {
        id: 'port-conflicts',
        question: 'Port 3000 or 8080 is already in use.',
        blocks: [
          {
            type: 'p',
            text: 'Stop the conflicting service or remap the published ports in your compose file (e.g. `"3100:80"` for the frontend, `"8180:8080"` for the API), then restart the containers.',
          },
        ],
      },
    ],
  },

  // ───────────────────────── Contributing ─────────────────────────
  {
    id: 'contributing',
    title: 'Contributing',
    icon: GitBranch,
    items: [
      {
        id: 'how-to-contribute',
        question: 'How do I contribute code?',
        blocks: [
          {
            type: 'links',
            links: [
              {
                label: 'Roadmap',
                href: 'https://github.com/orgs/NoFxAiOS/projects/3',
              },
              {
                label: 'Task Dashboard',
                href: 'https://github.com/orgs/NoFxAiOS/projects/5',
              },
              {
                label: 'CONTRIBUTING.md',
                href: 'https://github.com/NoFxAiOS/nofx/blob/dev/CONTRIBUTING.md',
              },
            ],
          },
          {
            type: 'steps',
            items: [
              'Pick a task from the boards above (filter by good first issue / help wanted) and comment "assign me".',
              'Fork the repo and branch from `dev`: `git checkout -b feat/your-topic`.',
              'Follow Conventional Commits; run `npm --prefix web run lint && npm --prefix web run build` before pushing.',
              'Open a PR against `NoFxAiOS/nofx:dev`, reference the issue (`Closes #123`), and attach screenshots for UI changes.',
            ],
          },
        ],
      },
      {
        id: 'bounty-program',
        question: 'Is there a bounty program?',
        blocks: [
          {
            type: 'p',
            text: 'Yes — selected issues carry cash bounties, plus badges, priority review, and beta access for regular contributors.',
          },
          {
            type: 'links',
            links: [
              {
                label: 'Issues with bounty label',
                href: 'https://github.com/NoFxAiOS/nofx/labels/bounty',
              },
              {
                label: 'Bounty claim template',
                href: 'https://github.com/NoFxAiOS/nofx/blob/dev/.github/ISSUE_TEMPLATE/bounty_claim.md',
              },
            ],
          },
        ],
      },
      {
        id: 'report-bugs',
        question: 'How do I report a bug?',
        blocks: [
          {
            type: 'p',
            text: 'Open a GitHub issue with the template: what you did, what happened, backend logs (`docker logs nofx-trading`), and screenshots. For suspected security issues, please follow the responsible-disclosure notes in SECURITY.md instead of a public issue.',
          },
          {
            type: 'links',
            links: [
              {
                label: 'New issue',
                href: 'https://github.com/NoFxAiOS/nofx/issues/new/choose',
              },
              {
                label: 'SECURITY.md',
                href: 'https://github.com/NoFxAiOS/nofx/blob/dev/SECURITY.md',
              },
            ],
          },
        ],
      },
    ],
  },
]

// ───────────────────────── Simplified Chinese mirror ─────────────────────────
// Same ids, icons, hrefs, ordering, code spans and brand names as above — only
// human-readable copy (title, question, block text, list/steps items, link
// labels) is translated.
export const faqCategoriesZh: FAQCategory[] = [
  // ───────────────────────── 新手入门 ─────────────────────────
  {
    id: 'getting-started',
    title: '新手入门',
    icon: BookOpen,
    items: [
      {
        id: 'what-is-nofx',
        question: 'NOFX 是什么？',
        blocks: [
          {
            type: 'p',
            text: 'NOFX 是一套开源、自托管的 AI 交易终端。其核心模式是 NOFX Autopilot（自动驾驶）：一个 AI 智能体，它会读取 Claw402.ai 信号榜，借助 Signal Lab 与强平结构筛选候选标的，用原始 K 线确认时机，并在 Hyperliquid 上执行交易 —— 全程运行在你自己的机器上，你的密钥永不离开你的服务器。',
          },
          {
            type: 'p',
            text: '除了 Autopilot，你还可以在 Strategy Studio 中构建自定义策略、并排运行多个 AI 交易员，并在排行榜上对比它们的表现。',
          },
        ],
      },
      {
        id: 'what-do-i-need',
        question: '启动 Autopilot 之前需要准备什么？',
        blocks: [
          {
            type: 'p',
            text: '两个已充值的账户 —— Config 页面的引导式启动会带你完成两者：',
          },
          {
            type: 'list',
            items: [
              'AI 费用钱包：一个 Base 链上的 USDC 钱包，用于支付 AI 模型与市场数据调用费用。启动最低需 `1 USDC`。',
              '一个已开通交易授权的 Hyperliquid 账户，且至少有 `12 USDC` 可用作保证金。',
            ],
          },
          {
            type: 'p',
            text: '启动按钮会先运行一次服务端预检，逐项检查前提条件，并精确指出缺失的环节，因此你不可能启动一个配置不全的机器人。',
          },
        ],
      },
      {
        id: 'which-markets',
        question: '它可以交易哪些市场？',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot 交易 Hyperliquid 永续合约：加密主流币（BTC、ETH、SOL……）外加覆盖美股、指数、大宗商品与外汇的 xyz 合成市场 —— 一个账户即可让 AI 在多种资产间运作。',
          },
          {
            type: 'p',
            text: '在 Strategy Studio 中手动搭建的交易员，还可以连接 Binance、Bybit、OKX、Bitget、KuCoin、Gate、Aster 与 Lighter。',
          },
        ],
      },
      {
        id: 'ai-models',
        question: '它使用哪些 AI 模型？我需要 API Key 吗？',
        blocks: [
          {
            type: 'p',
            text: '无需任何 API Key。NOFX 通过 Claw402 按量付费的基础设施完成推理：你的 AI 费用钱包以 Base 链 USDC 按次付费，终端按需调用受支持的模型（DeepSeek 及其他前沿模型）。',
          },
          {
            type: 'p',
            text: '高级用户仍可在 Config → Models 下接入自己的供应商密钥（OpenAI、Claude、Gemini、DeepSeek、Qwen、Grok、Kimi，或任何兼容 OpenAI 的接口）。',
          },
        ],
      },
      {
        id: 'is-it-profitable',
        question: '它能帮我赚钱吗？',
        blocks: [
          {
            type: 'p',
            text: '没有人能对此做出承诺，你也应警惕任何做出承诺的人。AI 遵循一套系统化的流程交易，但市场充满对抗性，过往表现绝不保证未来结果。',
          },
          {
            type: 'p',
            text: '看板刻意对绩效保持诚实：它区分已实现与未实现盈亏，展示手续费拖累链条（毛收益 − 手续费 = 净收益）、盈利因子，以及基于你真实起始余额计算的最大回撤。请关注这些数字，从小资金起步，只用亏得起的闲钱交易。',
          },
          {
            type: 'note',
            text: '交易存在重大亏损风险。NOFX 是软件，而非投资建议。',
          },
        ],
      },
    ],
  },

  // ───────────────────────── 启动与钱包 ─────────────────────────
  {
    id: 'launch-wallets',
    title: '启动与钱包',
    icon: Zap,
    items: [
      {
        id: 'ai-fee-wallet',
        question: '什么是 AI 费用钱包？',
        blocks: [
          {
            type: 'p',
            text: '一个专用于 Base 链的 EVM 钱包，用来支付 AI 模型调用与付费市场数据（x402 微支付）。它与你的交易保证金完全隔离 —— 永远不会接触 Hyperliquid。',
          },
          {
            type: 'list',
            items: [
              '引导式设置会为你创建（或复用已有的）钱包。',
              '仅向该地址在 Base 网络上转入 USDC。',
              '启动至少需要 `1 USDC`；充值后余额显示会自动刷新。',
              '一次典型循环的费用从不到一分到几美分不等，取决于所使用的模型。',
            ],
          },
        ],
      },
      {
        id: 'fee-wallet-private-key',
        question: 'AI 费用钱包的私钥保存在哪里？',
        blocks: [
          {
            type: 'p',
            text: '该私钥在你的服务器本地生成，以加密形式（AES-256）存储在你的自有数据库中，并在引导屏幕上仅向你展示一次。请务必备份 —— 一旦数据库丢失，密钥无法恢复。',
          },
          {
            type: 'note',
            text: '只在该钱包中存放费用资金。它的作用是支付 AI 调用费用，而非储蓄。',
          },
        ],
      },
      {
        id: 'hyperliquid-authorization',
        question: 'Hyperliquid 授权是如何工作的？安全吗？',
        blocks: [
          {
            type: 'p',
            text: 'NOFX 使用 Hyperliquid 代理钱包，因此你的主钱包私钥绝不会被共享。连接流程包含四个签名步骤：',
          },
          {
            type: 'steps',
            items: [
              '连接你的 EVM 钱包（Rabby、MetaMask、OKX、Coinbase Wallet）。',
              '授权一个新生成的 NOFX 代理钱包 —— 有效期 180 天，仅限交易。',
              '授权 builder 费用（每笔订单的小额平台费用）。',
              '将代理密钥保存到你的 NOFX 服务器（加密存储）。',
            ],
          },
          {
            type: 'p',
            text: '代理钱包只能下单与平仓，别无他权。它无法提取资金，你的保证金始终留在你自己的 Hyperliquid 账户中。',
          },
        ],
      },
      {
        id: 'launch-preflight',
        question: '启动预检会检查哪些内容？',
        blocks: [
          {
            type: 'p',
            text: '在任何创建或变更发生之前，服务端会用实时数据校验完整的链路：',
          },
          {
            type: 'list',
            items: [
              'AI 模型已启用并配置了凭据。',
              'AI 费用钱包密钥有效，且 Base 链 USDC 余额至少为 `1 USDC`（链上查询）。',
              'Hyperliquid 账户已完成授权（代理 + builder 费用）且可连通。',
              '交易资金：计入持仓权益后至少为 `12 USDC`。',
            ],
          },
          {
            type: 'p',
            text: '每一项未通过的检查都会明确指出修复方式，并深度链接到对应的引导设置。相同的检查在每次启动时都会于服务端强制执行，因此界面无法被意外绕过。',
          },
        ],
      },
      {
        id: 'relaunch-behavior',
        question: '如果我再次点击启动会怎样？',
        blocks: [
          {
            type: 'p',
            text: '启动是幂等的。如果 NOFX Autopilot 已存在，启动器会用当前的策略配置更新它并重新启动 —— 绝不会创建重复实例。如果机器人正处于循环中，重启最多需要约一分钟；界面会等待其完成。',
          },
        ],
      },
      {
        id: 'deposit-not-showing',
        question: '我已经充值了 USDC，但余额仍显示为零。',
        blocks: [
          {
            type: 'list',
            items: [
              'AI 费用钱包：请确认你是在 Base 网络上向界面显示的准确地址转入 USDC。余额会被缓存约 30 秒，设置面板每几秒会自动重新查询一次。',
              'Hyperliquid：充值会进入你自己的 Hyperliquid 账户；余额步骤会轮询账户实时状态。如有疑问，请在引导面板中点击 Refresh。',
              '如果链上 RPC 暂时不可达，面板会将余额标记为“未知”而非 0 —— 请稍候一分钟重试。',
            ],
          },
        ],
      },
    ],
  },

  // ───────────────────────── 交易与执行 ─────────────────────────
  {
    id: 'trading',
    title: '交易与执行',
    icon: TrendingUp,
    items: [
      {
        id: 'autopilot-pipeline',
        question: 'Autopilot 策略是如何一步步工作的？',
        blocks: [
          {
            type: 'p',
            text: '每个循环都会运行相同的四阶段漏斗 —— 任一阶段都可能淘汰某个候选标的，只有全部通过四关的设定才会被交易：',
          },
          {
            type: 'steps',
            items: [
              '构建标的池 —— 拉取 Claw402.ai 实时榜单，选取加密主流币与 xyz 合成市场（美股、指数、大宗商品、外汇）中的头部候选（默认 10 个），每个都带有方向偏好与信号 z 分数。',
              '验证每个候选 —— 获取其 Signal Lab 深度信号，以及价格附近的成本/强平结构：强平密集区与成本基线揭示了一次行情是前方有燃料，还是挡着高墙。',
              '确认时机 —— 读取原始 15 分钟 OHLCV K 线（30 根），确认入场是顺势借力，而非追高已拉伸的行情。',
              '决策与仓位 —— 只有越过置信度阈值（默认 `78/100`）且风险回报比约 `3:1` 的设定，才以 10 倍杠杆开仓；平仓始终先于开仓执行，且每个循环都会同时考虑多头与空头两个方向。',
            ],
          },
          {
            type: 'p',
            text: '第五层完全独立于 AI：硬性风控（持仓数量上限、杠杆上限、保证金上限、交易节流）会否决任何违反它们的决策，无论模型有多自信。',
          },
        ],
      },
      {
        id: 'data-sources',
        question: '它使用哪些数据？哪些部分是付费的？',
        blocks: [
          {
            type: 'list',
            items: [
              'Claw402.ai 信号数据 —— 排行榜、各币种 Signal Lab 深度信号，以及市场净流入。这些是付费接口，按次以 USDC 从你的 AI 费用钱包计费（x402 微支付）。',
              '成本/强平热力图 —— 每个市场的持仓成本与强平密集区聚合结构。',
              'Hyperliquid 行情数据 —— 原始 OHLCV K 线与实时 L2 订单簿（免费公开数据）。',
              '通过代理钱包读取你的账户 —— 净值、可用保证金，以及带盈亏的持仓。',
              '自身的交易历史 —— 已平仓交易会把胜率、盈利因子与回撤反馈到下一轮提示词中，让 AI 了解自己近期的状态。',
            ],
          },
          {
            type: 'note',
            text: '看板会以较慢的频率（每几分钟）轮询付费的 Claw402 接口，以节省你的费用钱包 —— 行情面板始终由免费数据源保持实时。',
          },
        ],
      },
      {
        id: 'decision-cycle',
        question: 'AI 多久做一次决策？',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot 每 5–15 分钟运行一次扫描循环，具体取决于你的启动方式（可按交易员配置，最短 3 分钟）。首个循环在启动后立即开始；单次循环通常耗时 30–60 秒，因为 AI 在决策前会读取完整的市场上下文。',
          },
        ],
      },
      {
        id: 'what-ai-sees',
        question: '每个循环 AI 都能看到哪些信息？',
        blocks: [
          {
            type: 'list',
            items: [
              '你的账户：净值、可用保证金，以及带盈亏的持仓。',
              'Claw402 排行榜：带有方向偏好的候选标的池。',
              '各候选的 Signal Lab 深度信号与成本/强平结构。',
              '用于确认时机的原始 OHLCV K 线。',
              '自身的战绩：胜率、盈利因子、回撤、近期交易。',
            ],
          },
          {
            type: 'p',
            text: '每个循环都会作为一条决策记录被保存 —— 看板上的执行日志会展示推理链条、所采取的动作，以及任何被拦截的订单。',
          },
        ],
      },
      {
        id: 'leverage-and-risk',
        question: '它使用怎样的杠杆与风险控制？',
        blocks: [
          {
            type: 'p',
            text: 'Autopilot 默认使用 10 倍全仓保证金。硬性风控运行在 AI 之外，无法被 AI 覆盖：',
          },
          {
            type: 'list',
            items: [
              '来自策略配置的持仓数量上限 —— 达到上限后将拒绝新的开仓。',
              '按资产类别划分的杠杆上限（BTC/ETH 与山寨币分别限制）。',
              '交易节流会阻止频繁折腾，例如在开仓后几分钟内平掉几乎没有波动的仓位。',
              '安全模式（见下文）会在 AI 自身失灵时保护整个仓位组合。',
            ],
          },
        ],
      },
      {
        id: 'safe-mode',
        question: '什么是安全模式？',
        blocks: [
          {
            type: 'p',
            text: '如果 AI 连续 3 个循环失败（供应商宕机、费用钱包余额为空、响应异常），交易员会进入安全模式：不再开新仓，已有持仓保持原有保护，循环持续重试。下一次 AI 调用成功时，会自动退出安全模式。',
          },
          {
            type: 'p',
            text: '安全模式会在看板上以横幅形式展示，并附带原因，因此绝不会悄无声息地发生。',
          },
        ],
      },
      {
        id: 'fee-wallet-empty-mid-run',
        question: '如果运行中 AI 费用钱包余额耗尽会怎样？',
        blocks: [
          {
            type: 'p',
            text: 'AI 调用将开始失败，并给出明确的“资金不足”状态。看板会显示一个持续的红色横幅，标注钱包余额；连续三个循环失败后，机器人进入安全模式。向费用钱包补充 Base 链 USDC 后，交易员会自动恢复 —— 无需重启。',
          },
        ],
      },
      {
        id: 'trading-fees',
        question: '我需要支付哪些费用？',
        blocks: [
          {
            type: 'list',
            items: [
              '每笔订单的 Hyperliquid 交易手续费，外加已授权的 builder 费用。',
              '从费用钱包按次支付的 AI/数据成本（每个循环几美分）。',
            ],
          },
          {
            type: 'p',
            text: '手续费是高频策略的隐形杀手。看板的统计条会展示完整的链条 —— 已实现毛盈亏，减去手续费，等于净利 —— 让你立刻看清手续费是否正在吞噬你的优势。',
          },
        ],
      },
      {
        id: 'stop-and-manual',
        question: '如何停止机器人或手动平仓？',
        blocks: [
          {
            type: 'list',
            items: [
              '停止：使用 Config 页面交易员列表中的 Stop 按钮。停止会中止决策循环；已开持仓保持开放，由你自行管理。',
              '手动平仓：在看板持仓面板中平掉任意仓位 —— 手动平仓会同步回仓位历史。',
              '紧急情况：你随时可以直接在 Hyperliquid 上管理持仓；NOFX 永远不会把你锁在自己的账户之外。',
            ],
          },
        ],
      },
    ],
  },

  // ───────────────────────── 看板与指标 ─────────────────────────
  {
    id: 'dashboard',
    title: '看板与指标',
    icon: Monitor,
    items: [
      {
        id: 'metrics-meaning',
        question: '顶部各项指标究竟代表什么？',
        blocks: [
          {
            type: 'list',
            items: [
              '净值 —— 包含未实现盈亏的实时账户价值。',
              '总盈亏（含未实现）—— 净值相对于起始余额的差额；随持仓波动。',
              '已实现盈亏（已平仓交易）—— 仅统计已结束交易的净结果；胜率、盈利因子与夏普比率都基于它计算。',
              '盈利因子 —— 已平仓交易的毛利 ÷ 毛亏；大于 1.0 表示已平仓组合整体为正。',
              '最大回撤 —— 已实现净值曲线从峰值到谷值的最大跌幅，以你的真实起始余额为基准衡量。',
            ],
          },
        ],
      },
      {
        id: 'pl-contradiction',
        question: '为什么总盈亏为正，而已实现盈亏却为负？',
        blocks: [
          {
            type: 'p',
            text: '两者衡量的是不同的东西。已实现盈亏只统计已平仓交易；总盈亏还包含仍在持仓中的未实现收益。一个机器人可能在已平仓交易上亏损，但其未平仓组合带来了足够的未实现利润，使总盈亏翻红 —— 反之亦然。查看“毛收益/手续费/净利”统计条，可以看清已实现结果中有多少是被手续费拖累的。',
          },
        ],
      },
      {
        id: 'execution-log',
        question: '在哪里可以看到 AI 为什么做了（或拒绝做）某件事？',
        blocks: [
          {
            type: 'p',
            text: '执行日志面板会列出每个循环所采取的动作、AI 调用耗时，以及任何被拦截的订单及其触发的确切风控（节流、持仓上限、风控）。完整的推理链条会随每条决策记录一同保存。',
          },
        ],
      },
      {
        id: 'competition',
        question: '排行榜 / 竞赛是什么？',
        blocks: [
          {
            type: 'p',
            text: '开启了“显示在竞赛中”的交易员会出现在公开排行榜上，按实时表现排名。此为按交易员可选的加入项，可随时在交易员列表中切换。',
          },
        ],
      },
    ],
  },

  // ───────────────────────── 安全 ─────────────────────────
  {
    id: 'security',
    title: '安全',
    icon: Shield,
    items: [
      {
        id: 'key-storage',
        question: '我的密钥是如何存储的？',
        blocks: [
          {
            type: 'list',
            items: [
              '所有密钥类数据（代理密钥、费用钱包私钥、交易所 API Key）都以 AES-256 在静止状态下加密，存储于你自己的数据库中。',
              '可选的 RSA 传输加密，保护浏览器与服务器之间传输中的密钥。',
              'NOFX 是自托管的：任何数据都不会发往任何第三方服务器。代码开源，可供审计。',
            ],
          },
        ],
      },
      {
        id: 'can-nofx-steal-funds',
        question: 'NOFX 能提取或盗取我的资金吗？',
        blocks: [
          {
            type: 'p',
            text: '不能。在 Hyperliquid 上，NOFX 永远只持有代理钱包，而代理钱包按协议设计只能交易、不能提现。你的保证金始终留在你自己的账户中，由你的主钱包掌控。',
          },
          {
            type: 'note',
            text: '如果你连接的是 CEX（中心化交易所），请只为其 API Key 授予交易权限 —— 关闭提现功能并设置 IP 白名单。',
          },
        ],
      },
      {
        id: 'registration-model',
        question: '为什么其他人无法在我的实例上注册？',
        blocks: [
          {
            type: 'p',
            text: '出于设计，一个实例仅限单一操作者：首个注册的账户成为操作者，注册随即关闭（“System already initialized”）。这可防止陌生人在暴露的部署上创建账户。每位操作者请运行一个独立实例。',
          },
        ],
      },
    ],
  },

  // ───────────────────────── 自托管与故障排查 ─────────────────────────
  {
    id: 'self-hosting',
    title: '自托管与故障排查',
    icon: Wrench,
    items: [
      {
        id: 'how-to-install',
        question: '如何安装 NOFX？',
        blocks: [
          {
            type: 'p',
            text: '在 Linux/macOS 上只需一行命令（通过 Docker 安装并启动全部服务）：',
          },
          {
            type: 'list',
            items: [
              '脚本方式：`curl -fsSL https://raw.githubusercontent.com/NoFxAiOS/nofx/main/install.sh | bash`',
              'Docker：下载 `docker-compose.prod.yml` 并运行 `docker compose -f docker-compose.prod.yml up -d`',
              'Windows：安装 Docker Desktop，然后使用上述 Docker 方式。',
              '源码编译：需 Go 1.21+、Node 18+、TA-Lib（`brew install ta-lib` / `apt-get install libta-lib0-dev`），随后运行 `go run .` 与 `npm --prefix web run dev`。',
            ],
          },
          {
            type: 'p',
            text: '随后打开 `http://127.0.0.1:3000` —— Web 界面使用 3000 端口，API 使用 8080 端口。',
          },
        ],
      },
      {
        id: 'how-to-update',
        question: '如何更新？',
        blocks: [
          {
            type: 'p',
            text: '重新运行安装脚本，或使用 Docker：`docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml up -d`。你的数据库与密钥位于挂载的 `data/` 目录中，更新后不会丢失。后端恢复后，正在运行的交易员会自动重启。',
          },
        ],
      },
      {
        id: 'launch-blocked',
        question: '启动被某项失败检查拦截了，该怎么办？',
        blocks: [
          {
            type: 'p',
            text: '请阅读提示信息：每一项预检失败都会指出修复方式，并引导你前往对应的设置步骤 —— 为 AI 钱包充值、完成 Hyperliquid 授权，或存入交易用的 USDC。余额会被实时重新校验，因此一旦你修复该项，启动即可通过。',
          },
        ],
      },
      {
        id: 'exchange-unreachable',
        question: '交易所账户显示“invalid credentials”或“unavailable”。',
        blocks: [
          {
            type: 'list',
            items: [
              '凭证无效：代理授权已过期（180 天）或保存的密钥已失效 —— 重新连接 Hyperliquid 钱包；流程中提供一键续期。',
              '不可用：交易所 API 未响应；账户状态会缓存 30 秒，请等待后刷新。',
              'CEX 密钥：请核实交易权限、IP 白名单，以及是否已开启期货/永续交易权限。',
            ],
          },
        ],
      },
      {
        id: 'where-are-logs',
        question: '日志在哪里？',
        blocks: [
          {
            type: 'list',
            items: [
              '后端：`docker logs nofx-trading`（或运行 `go run .` 的终端）。',
              '每个循环的 AI 推理与错误：看板执行日志。',
              '前端构建/运行时问题：浏览器开发者工具控制台。',
            ],
          },
        ],
      },
      {
        id: 'port-conflicts',
        question: '端口 3000 或 8080 已被占用。',
        blocks: [
          {
            type: 'p',
            text: '停止冲突的服务，或在 compose 文件中重新映射对外端口（例如前端用 `"3100:80"`、API 用 `"8180:8080"`），然后重启容器。',
          },
        ],
      },
    ],
  },

  // ───────────────────────── 参与贡献 ─────────────────────────
  {
    id: 'contributing',
    title: '参与贡献',
    icon: GitBranch,
    items: [
      {
        id: 'how-to-contribute',
        question: '如何参与代码贡献？',
        blocks: [
          {
            type: 'links',
            links: [
              {
                label: '路线图',
                href: 'https://github.com/orgs/NoFxAiOS/projects/3',
              },
              {
                label: '任务看板',
                href: 'https://github.com/orgs/NoFxAiOS/projects/5',
              },
              {
                label: '贡献指南（CONTRIBUTING.md）',
                href: 'https://github.com/NoFxAiOS/nofx/blob/dev/CONTRIBUTING.md',
              },
            ],
          },
          {
            type: 'steps',
            items: [
              '从上述看板中挑选一个任务（按 good first issue / help wanted 筛选），并评论“assign me”（分配给我）。',
              'Fork 仓库，并从 `dev` 分支切出新分支：`git checkout -b feat/your-topic`。',
              '遵循 Conventional Commits 规范；推送前运行 `npm --prefix web run lint && npm --prefix web run build`。',
              '向 `NoFxAiOS/nofx:dev` 发起 PR，引用对应 issue（`Closes #123`），UI 改动请附上截图。',
            ],
          },
        ],
      },
      {
        id: 'bounty-program',
        question: '有赏金计划吗？',
        blocks: [
          {
            type: 'p',
            text: '有 —— 部分入选 issue 提供现金赏金，此外还有徽章、优先评审，以及为长期贡献者提供的 Beta 访问权限。',
          },
          {
            type: 'links',
            links: [
              {
                label: '带有 bounty 标签的 issue',
                href: 'https://github.com/NoFxAiOS/nofx/labels/bounty',
              },
              {
                label: '赏金认领模板',
                href: 'https://github.com/NoFxAiOS/nofx/blob/dev/.github/ISSUE_TEMPLATE/bounty_claim.md',
              },
            ],
          },
        ],
      },
      {
        id: 'report-bugs',
        question: '如何报告一个 Bug？',
        blocks: [
          {
            type: 'p',
            text: '使用模板提交 GitHub issue：你做了什么、发生了什么、后端日志（`docker logs nofx-trading`），以及截图。如怀疑存在安全问题，请遵循 SECURITY.md 中的负责任披露说明，而非公开提交 issue。',
          },
          {
            type: 'links',
            links: [
              {
                label: '新建 issue',
                href: 'https://github.com/NoFxAiOS/nofx/issues/new/choose',
              },
              {
                label: 'SECURITY.md',
                href: 'https://github.com/NoFxAiOS/nofx/blob/dev/SECURITY.md',
              },
            ],
          },
        ],
      },
    ],
  },
]

/** Returns the FAQ content for the active language. */
export function getFaqCategories(language: Language): FAQCategory[] {
  return language === 'zh' ? faqCategoriesZh : faqCategoriesEn
}
