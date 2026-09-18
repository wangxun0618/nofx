import { Brain, Landmark, Rocket, Sparkles } from 'lucide-react'
import { t, type Language } from '../../i18n/translations'

interface BeginnerGuideCardsProps {
  language: Language
  claw402Ready: boolean
  exchangeReady: boolean
  strategyReady: boolean
  traderReady: boolean
  canCreateTrader: boolean
  walletAddress?: string | null
  onQuickSetupClaw402: () => void
  onOpenExchange: () => void
  onOpenStrategy: () => void
  onCreateTrader: () => void
}

function truncateAddress(address: string) {
  if (address.length <= 12) return address
  return `${address.slice(0, 6)}...${address.slice(-4)}`
}

export function BeginnerGuideCards({
  language,
  claw402Ready,
  exchangeReady,
  strategyReady,
  traderReady,
  canCreateTrader,
  walletAddress,
  onQuickSetupClaw402,
  onOpenExchange,
  onOpenStrategy,
  onCreateTrader,
}: BeginnerGuideCardsProps) {
  const isZh = language === 'zh'

  const cards = [
    {
      key: 'model',
      icon: Brain,
      title: t('onboarding.step1Title', language),
      desc: t('onboarding.step1Desc', language),
      meta: walletAddress
        ? isZh
          ? `Wallet ${truncateAddress(walletAddress)}`
          : `Wallet ${truncateAddress(walletAddress)}`
        : t('onboarding.payPerCall', language),
      ready: claw402Ready,
      actionLabel: claw402Ready
        ? t('status.configured', language)
        : t('onboarding.oneClickSetup', language),
      onAction: onQuickSetupClaw402,
      disabled: claw402Ready,
    },
    {
      key: 'exchange',
      icon: Landmark,
      title: t('onboarding.step2Title', language),
      desc: t('onboarding.step2Desc', language),
      meta: exchangeReady
        ? t('status.ready', language)
        : t('onboarding.exchangeOptions', language),
      ready: exchangeReady,
      actionLabel: exchangeReady
        ? t('status.manage', language)
        : t('status.configure', language),
      onAction: onOpenExchange,
      disabled: false,
    },
    {
      key: 'strategy',
      icon: Sparkles,
      title: t('onboarding.step3Title', language),
      desc: t('onboarding.step3Desc', language),
      meta: strategyReady
        ? t('onboarding.strategyReady', language)
        : t('onboarding.optionalWorthLook', language),
      ready: strategyReady,
      actionLabel: t('onboarding.openStrategy', language),
      onAction: onOpenStrategy,
      disabled: false,
    },
    {
      key: 'trader',
      icon: Rocket,
      title: t('onboarding.step4Title', language),
      desc: t('onboarding.step4Desc', language),
      meta: traderReady
        ? t('onboarding.traderCreated', language)
        : canCreateTrader
          ? t('onboarding.readyToCreate', language)
        : t('onboarding.finishFirstThree', language),
      ready: traderReady,
      actionLabel: traderReady
        ? t('onboarding.createAnother', language)
        : t('onboarding.createNow', language),
      onAction: onCreateTrader,
      disabled: !canCreateTrader,
    },
  ]

  return (
    <section className="space-y-4 rounded-[28px] border border-nofx-gold/20 bg-nofx-bg-lighter p-5">
      <div className="flex items-center justify-between gap-4">
        <div>
          <div className="text-xs font-semibold uppercase tracking-[0.3em] text-nofx-gold/80">
            {t('onboarding.quickstart', language)}
          </div>
          <h2 className="mt-1 text-xl font-bold text-nofx-text">
            {t('onboarding.followSteps', language)}
          </h2>
        </div>
        {/* <div className="rounded-full border border-nofx-gold/20 bg-nofx-bg-deeper px-3 py-1 text-xs text-nofx-text-muted">
          Hidden in advanced mode
        </div> */}
      </div>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {cards.map((card) => {
          const Icon = card.icon
          return (
            <div
              key={card.key}
              className="rounded-[22px] border border-nofx-gold/20 bg-nofx-bg-deeper p-4"
            >
              <div className="flex items-center justify-between gap-3">
                <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-nofx-gold/10 text-nofx-gold">
                  <Icon className="h-5 w-5" />
                </div>
                <span
                  className={`rounded-full px-2.5 py-1 text-[10px] font-bold uppercase tracking-[0.22em] ${
                    card.ready
                      ? 'bg-nofx-success/15 text-nofx-success'
                      : 'bg-nofx-bg-deeper text-nofx-text-muted'
                  }`}
                >
                  {card.ready
                    ? t('status.ready', language)
                    : t('status.pending', language)}
                </span>
              </div>

              <h3 className="mt-4 text-base font-semibold text-nofx-text">
                {card.title}
              </h3>
              <p className="mt-2 min-h-[72px] text-sm leading-6 text-nofx-text-muted">
                {card.desc}
              </p>
              <div className="mt-3 text-xs text-nofx-text-muted">{card.meta}</div>

              <button
                type="button"
                onClick={card.onAction}
                disabled={card.disabled}
                className={`mt-5 w-full rounded-2xl px-4 py-3 text-sm font-semibold transition ${
                  card.disabled
                    ? 'cursor-not-allowed bg-nofx-bg-deeper text-nofx-text-muted'
                    : 'bg-nofx-gold text-white hover:bg-nofx-gold/90'
                }`}
              >
                {card.actionLabel}
              </button>
            </div>
          )
        })}
      </div>
    </section>
  )
}
