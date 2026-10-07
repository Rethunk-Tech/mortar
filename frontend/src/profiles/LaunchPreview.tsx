import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Typography } from '@mui/material'
import { Copy } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import type { CommandPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { PreviewCommand } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { copyText } from '../share/copyText.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { errorMessage } from '../toasts/report.ts'

const PREVIEW_DEBOUNCE_MS = 300

function shellQuote(value: string): string {
  return `'${value.replaceAll("'", "'\\''")}'`
}

function formatPreview(preview: CommandPreview): string {
  return [...(preview.env ?? []), (preview.argv ?? []).map(shellQuote).join(' ')].join('\n')
}

function formatShellLine(preview: CommandPreview): string {
  return [...(preview.env ?? []), (preview.argv ?? []).map(shellQuote).join(' ')].join(' ')
}

export function LaunchPreview({
  gameId,
  profileId,
  options,
  prefix,
  env,
}: {
  gameId: string
  profileId: string
  options: string
  prefix: string
  env: string
}) {
  const { t } = useLingui()
  const [preview, setPreview] = useState<CommandPreview>({ env: [], argv: [], error: '' })
  const [loadError, setLoadError] = useState('')
  const read = useCallback(
    () =>
      PreviewCommand(gameId, profileId, options, prefix, env)
        .then((next) => {
          setLoadError('')
          setPreview(next)
        })
        .catch((e: unknown) => setLoadError(errorMessage(e))),
    [env, gameId, options, prefix, profileId],
  )
  useEffect(() => {
    const timer = setTimeout(read, PREVIEW_DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [read])
  const copyCommand = () => {
    copyText(formatShellLine(preview), t`Command copied`)
  }

  return (
    <Box sx={{ mt: 1, mb: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
        <Typography sx={{ fontSize: 12, color: 'text.secondary', flex: 1 }}>
          {t`Direct launch preview`}
        </Typography>
        {preview.error || loadError ? null : (
          <IconButton
            size="small"
            aria-label={t`Copy the command`}
            onClick={copyCommand}
            sx={{ color: 'text.secondary' }}
          >
            <Copy size={14} />
          </IconButton>
        )}
      </Box>
      {preview.error || loadError ? (
        <Typography color="error" sx={{ fontSize: 12 }}>
          {preview.error || loadError}
          {loadError && !preview.error ? (
            <Button size="small" onClick={read}>
              {t`Retry`}
            </Button>
          ) : null}
        </Typography>
      ) : (
        <Box
          component="pre"
          sx={{
            m: 0,
            p: space.gap,
            // Wrapped rather than scrolled sideways: a scroll region would need its own keyboard stop.
            whiteSpace: 'pre-wrap',
            overflowWrap: 'anywhere',
            color: 'text.secondary',
            fontFamily: MONO,
            fontSize: 12,
            userSelect: 'text',
          }}
        >
          {formatPreview(preview)}
        </Box>
      )}
    </Box>
  )
}
