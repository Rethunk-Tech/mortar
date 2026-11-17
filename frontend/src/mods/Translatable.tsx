import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { type ReactNode, useState } from 'react'
import { TranslateDescription } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { type InlineError, inlineError } from '../toasts/report.ts'

/** Optional live translation of source-site copy into the player's language. Each instance is its own toggle. */
export function Translatable({
  source,
  children,
}: {
  source: string
  children: (text: string) => ReactNode
}) {
  const { i18n } = useLingui()
  return (
    <TranslatableView key={`${source}\0${i18n.locale}`} source={source} locale={i18n.locale}>
      {children}
    </TranslatableView>
  )
}

function TranslatableView({
  source,
  locale,
  children,
}: {
  source: string
  locale: string
  children: (text: string) => ReactNode
}) {
  const { t } = useLingui()
  const [shown, setShown] = useState<'orig' | 'translated'>('orig')
  const [translated, setTranslated] = useState<string | null>(null)
  const [error, setError] = useState<InlineError | null>(null)
  const [busy, setBusy] = useState(false)
  const showTranslated = shown === 'translated' && translated !== null
  const run = () => {
    if (translated !== null) {
      setShown('translated')
      return
    }
    setBusy(true)
    setError(null)
    TranslateDescription(source, locale)
      .then((out) => {
        setTranslated(out)
        setShown('translated')
      })
      .catch((e: unknown) => {
        setError(inlineError(e))
      })
      .finally(() => {
        setBusy(false)
      })
  }
  if (source === '') {
    return children(source)
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Button
        size="small"
        disabled={busy}
        onClick={() => {
          if (showTranslated) {
            setShown('orig')
            return
          }
          run()
        }}
        sx={{ alignSelf: 'flex-start', px: 0.5, fontSize: 'small.fontSize' }}
      >
        {showTranslated ? t`Show original` : t`Translate`}
      </Button>
      {error ? <ErrorRetry error={error} onRetry={run} /> : null}
      {children(showTranslated ? translated : source)}
    </Box>
  )
}
