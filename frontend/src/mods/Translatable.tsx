import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import { TranslateDescription } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { inlineError, type InlineError } from '../toasts/report.ts'

/** Optional live translation of source-site copy into the player's language. Each instance is its own toggle. */
export function Translatable({
  source,
  children,
}: {
  source: string
  children: (text: string) => ReactNode
}) {
  const { t, i18n } = useLingui()
  const locale = i18n.locale
  const [shown, setShown] = useState<'orig' | 'translated'>('orig')
  const [translated, setTranslated] = useState<string | null>(null)
  const [error, setError] = useState<InlineError | null>(null)
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    setShown('orig')
    setTranslated(null)
    setError(null)
  }, [source, locale])
  const showTranslated = shown === 'translated' && translated !== null
  const run = async () => {
    if (translated !== null) {
      setShown('translated')
      return
    }
    setBusy(true)
    setError(null)
    try {
      setTranslated(await TranslateDescription(source, locale))
      setShown('translated')
    } catch (e) {
      setError(inlineError(e))
    } finally {
      setBusy(false)
    }
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
          void run()
        }}
        sx={{ alignSelf: 'flex-start', px: 0.5, fontSize: 'small.fontSize' }}
      >
        {showTranslated ? t`Show original` : t`Translate`}
      </Button>
      {error ? <ErrorRetry error={error} onRetry={() => void run()} /> : null}
      {children(showTranslated ? translated : source)}
    </Box>
  )
}
