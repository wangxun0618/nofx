import { motion } from 'framer-motion'
import { Terminal, Cpu, Share2, Shield, Activity, Code } from 'lucide-react'
import { t } from '../../../i18n/translations'
import { useLanguage } from '../../../contexts/LanguageContext'

const features = [
    {
        icon: Terminal,
        titleKey: 'landing2.featureAiDriven',
        descKey: 'landing2.featureAiDrivenDesc',
    },
    {
        icon: Cpu,
        titleKey: 'landing2.featureAutonomous',
        descKey: 'landing2.featureAutonomousDesc',
    },
    {
        icon: Share2,
        titleKey: 'landing2.featurePunkSocial',
        descKey: 'landing2.featurePunkSocialDesc',
    },
    {
        icon: Shield,
        titleKey: 'landing2.featureNonCustodial',
        descKey: 'landing2.featureNonCustodialDesc',
    },
    {
        icon: Activity,
        titleKey: 'landing2.featureHighFrequency',
        descKey: 'landing2.featureHighFrequencyDesc',
    },
    {
        icon: Code,
        titleKey: 'landing2.featureOpenSource',
        descKey: 'landing2.featureOpenSourceDesc',
    }
]

export default function BrandFeatures() {
    const { language } = useLanguage()
    return (
        <section id="features" className="py-24 bg-nofx-bg relative">
            <div className="max-w-[1920px] mx-auto px-6 lg:px-16">

                <div className="mb-16 border-l-4 border-nofx-gold pl-6">
                    <h2 className="text-4xl md:text-5xl font-black text-nofx-text uppercase tracking-tighter mb-4">
                        {t('landing2.coreProtocol', language)}{' '}
                        <span className="text-nofx-text-muted">
                            {t('landing2.specs', language)}
                        </span>
                    </h2>
                    <p className="text-xl text-nofx-text-muted font-mono">
                        {t('landing2.specsSubtitle', language)}
                    </p>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-1">
                    {features.map((f, i) => (
                        <motion.div
                            key={i}
                            className="group relative bg-nofx-bg-lighter border border-[rgba(26,24,19,0.14)] p-8 hover:bg-nofx-bg-deeper transition-colors cursor-default overflow-hidden"
                            initial={{ opacity: 0, y: 20 }}
                            whileInView={{ opacity: 1, y: 0 }}
                            viewport={{ once: true }}
                            transition={{ delay: i * 0.1 }}
                        >
                            <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
                                <f.icon size={100} />
                            </div>

                            <f.icon className="w-10 h-10 text-nofx-gold mb-6" />

                            <h3 className="text-xl font-bold text-nofx-text mb-3 uppercase flex items-center gap-2">
                                {t(f.titleKey, language)}
                            </h3>

                            <p className="text-nofx-text-muted leading-relaxed text-sm md:text-base">
                                {t(f.descKey, language)}
                            </p>

                            <div className="absolute bottom-0 left-0 w-full h-1 bg-nofx-gold transform scale-x-0 group-hover:scale-x-100 transition-transform origin-left duration-300" />
                        </motion.div>
                    ))}
                </div>
            </div>
        </section>
    )
}
