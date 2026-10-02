import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { CommandPreview } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { PreviewCommand } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'

const PREVIEW_DEBOUNCE_MS = 300

function shellQuote(value: string): string {
  return `'${value.replaceAll("'", "'\\''")}'`
}

function formatPreview(preview: CommandPreview): string {
  return [...(preview.env ?? []), (preview.argv ?? []).map(shellQuote).join(' ')].join('\n')
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
  useEffect(() => {
    let active = true
    const timer = setTimeout(
      () =>
        PreviewCommand(gameId, profileId, options, prefix, env)
          .then((next) => {
            if (active) {
              setPreview(next)
            }
          })
          .catch(() => undefined),
      PREVIEW_DEBOUNCE_MS,
    )
    return () => {
      active = false
      clearTimeout(timer)
    }
  }, [env, gameId, options, prefix, profileId])
  return (
    <Box sx={{ mt: 1, mb: 1 }}>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', mb: 0.5 }}>
        {t`Direct launch preview`}
      </Typography>
      {preview.error ? (
        <Typography color="error" sx={{ fontSize: 12 }}>
          {preview.error}
        </Typography>
      ) : (
        <Box
          component="pre"
          sx={{
            m: 0,
            p: 1,
            overflowX: 'auto',
            color: 'text.secondary',
            fontFamily: 'monospace',
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
