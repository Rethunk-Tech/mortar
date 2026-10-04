import { useLingui } from '@lingui/react/macro'
import { Box, IconButton, Typography } from '@mui/material'
import { Clipboard } from '@wailsio/runtime'
import { Copy } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { CommandPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { PreviewCommand } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { MONO } from '../theme/theme.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

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
  useEffect(() => {
    let active = true
    const timer = setTimeout(
      () =>
        PreviewCommand(gameId, profileId, options, prefix, env)
          .then((next) => {
            if (active) {
              setLoadError('')
              setPreview(next)
            }
          })
          .catch((e: unknown) => {
            if (active) {
              setLoadError(errorMessage(e))
            }
          }),
      PREVIEW_DEBOUNCE_MS,
    )
    return () => {
      active = false
      clearTimeout(timer)
    }
  }, [env, gameId, options, prefix, profileId])
  const copyCommand = () => {
    Clipboard.SetText(formatShellLine(preview)).then(
      () => useToasts.getState().push({ kind: 'success', title: t`Command copied` }),
      reportUnexpected,
    )
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
        </Typography>
      ) : (
        <Box
          component="pre"
          sx={{
            m: 0,
            p: 1,
            overflowX: 'auto',
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
