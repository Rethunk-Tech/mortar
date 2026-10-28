import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Typography } from '@mui/material'
import { Copy } from 'lucide-react'
import type { CommandPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { PreviewCommand } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { copyText } from '../share/copyText.ts'
import { useDebouncedLoad } from '../shell/useDebouncedLoad.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { errorMessage } from '../toasts/report.ts'

const PREVIEW_DEBOUNCE_MS = 300
const NO_PREVIEW: CommandPreview = { env: [], argv: [], error: '' }
const PREVIEW_LINE_PX = 18
const PREVIEW_HEIGHT_PX = 44

function shellQuote(value: string): string {
  return `'${value.replaceAll("'", "'\\''")}'`
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
  const load = () => PreviewCommand(gameId, profileId, options, prefix, env)
  const {
    data: preview,
    error,
    run,
  } = useDebouncedLoad(load, [gameId, profileId, options, prefix, env], {
    initial: NO_PREVIEW,
    delayMs: PREVIEW_DEBOUNCE_MS,
  })
  const loadError = error === null ? '' : errorMessage(error)
  const read = () => run(load)
  const copyCommand = () => {
    copyText(formatShellLine(preview), t`Command copied`)
  }

  return (
    <Box sx={{ mt: space.gap, mb: space.gap }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
        <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary', flex: 1 }}>
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
        <Typography color="error" sx={{ fontSize: 'small.fontSize' }}>
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
          // A scroll region needs its own keyboard stop.
          tabIndex={0}
          aria-label={t`Direct launch preview`}
          sx={{
            m: 0,
            p: space.gap,
            // One line at a fixed height, scrolled sideways, so typing never moves the controls below.
            whiteSpace: 'pre',
            overflowX: 'auto',
            overflowY: 'hidden',
            height: PREVIEW_HEIGHT_PX,
            lineHeight: `${PREVIEW_LINE_PX}px`,
            boxSizing: 'border-box',
            color: 'text.secondary',
            fontFamily: MONO,
            fontSize: 'small.fontSize',
            userSelect: 'text',
          }}
        >
          {formatShellLine(preview)}
        </Box>
      )}
    </Box>
  )
}
