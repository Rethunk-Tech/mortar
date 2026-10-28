import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import { useGameInfo } from '../games/info.ts'
import { openPage } from '../mods/menu.ts'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { graphicsOptions } from './graphicsAsk.ts'
import { cancelling } from './playModeState.ts'
import { useLaunch } from './store.ts'

export function GraphicsDialog() {
  const { t } = useLingui()
  const ask = useLaunch((s) => s.graphicsAsk)
  const cancel = useLaunch((s) => s.dismissGraphicsAsk)
  const answer = useLaunch((s) => s.answerGraphics)
  const quit = cancelling(cancel)
  const info = useGameInfo(ask?.game)
  const graphics = info?.graphics
  const name = info?.name ?? ''
  return (
    <AppDialog
      open={ask !== null}
      onClose={quit}
      size="sm"
      title={t`Choose a graphics API for ${name}`}
      footer={
        <DialogFooter>
          <Button onClick={quit}>{t`Cancel`}</Button>
          {graphics
            ? graphicsOptions(graphics)
                .reverse()
                .map((o) => (
                  <Button
                    key={o.id}
                    variant={o.recommended ? 'contained' : 'text'}
                    onClick={() => answer(o.id).catch(reportUnexpected)}
                  >
                    {o.recommended ? t`${{ name: o.label }} (recommended)` : o.label}
                  </Button>
                ))
            : null}
        </DialogFooter>
      }
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        {ask?.afterEarlyExit ? (
          <Typography sx={{ fontWeight: 600, color: 'text.secondary' }}>
            {t`The game closed right after starting last time.`}
          </Typography>
        ) : null}
        <Typography sx={{ color: 'text.secondary' }}>{graphics?.explanation}</Typography>
        {graphics ? (
          <Link
            component="button"
            variant="body2"
            sx={{ textAlign: 'left' }}
            onClick={() => openPage(graphics.source)}
          >
            {graphics.reason}
          </Link>
        ) : null}
        <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
          {t`You can change this later in ${name}'s settings, under Play.`}
        </Typography>
      </Box>
    </AppDialog>
  )
}
